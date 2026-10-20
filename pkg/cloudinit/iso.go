package cloudinit

import (
	"errors"
	"fmt"
	"os"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
)

const volumeLabel = "CIDATA"

// BuildISO 生成 label=CIDATA 的 cloud-init NoCloud seed ISO。
func BuildISO(config Config) ([]byte, error) {
	files := []struct {
		name    string
		content string
	}{
		{"meta-data", config.MetaData()},
		{"user-data", config.UserData()},
		{"network-config", config.NetworkConfig()},
	}

	total := 0
	for _, file := range files {
		total += len(file.content)
	}
	size := int64(64*1024 + total)
	size = (size + 2047) / 2048 * 2048

	tmp, err := os.CreateTemp("", "vm-control-seed-*.iso")
	if err != nil {
		return nil, fmt.Errorf("创建 seed 临时文件失败: %w", err)
	}
	path := tmp.Name()
	if err := tmp.Close(); err != nil {
		os.Remove(path)
		return nil, fmt.Errorf("关闭 seed 临时文件失败: %w", err)
	}
	os.Remove(path)
	defer os.Remove(path)

	diskImage, err := diskfs.Create(path, size, diskfs.SectorSizeDefault)
	if err != nil {
		return nil, fmt.Errorf("创建 seed 镜像失败: %w", err)
	}
	diskImage.LogicalBlocksize = 2048

	fs, err := diskImage.CreateFilesystem(disk.FilesystemSpec{
		Partition:   0,
		FSType:      filesystem.TypeISO9660,
		VolumeLabel: volumeLabel,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 seed ISO9660 文件系统失败: %w", err)
	}

	for _, file := range files {
		f, err := fs.OpenFile(file.name, os.O_CREATE|os.O_RDWR)
		if err != nil {
			_ = fs.Close()
			return nil, fmt.Errorf("创建 seed 文件 %s 失败: %w", file.name, err)
		}
		if _, err := f.Write([]byte(file.content)); err != nil {
			_ = f.Close()
			_ = fs.Close()
			return nil, fmt.Errorf("写入 seed 文件 %s 失败: %w", file.name, err)
		}
		if err := f.Close(); err != nil {
			_ = fs.Close()
			return nil, fmt.Errorf("关闭 seed 文件 %s 失败: %w", file.name, err)
		}
	}

	iso, ok := fs.(*iso9660.FileSystem)
	if !ok {
		_ = fs.Close()
		return nil, errors.New("seed 文件系统不是 ISO9660")
	}
	if err := iso.Finalize(iso9660.FinalizeOptions{
		VolumeIdentifier: volumeLabel,
		RockRidge:        true,
		Joliet:           true,
	}); err != nil {
		_ = fs.Close()
		return nil, fmt.Errorf("完成 seed ISO 失败: %w", err)
	}
	if err := fs.Close(); err != nil {
		return nil, fmt.Errorf("关闭 seed 文件系统失败: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 seed ISO 失败: %w", err)
	}
	return data, nil
}
