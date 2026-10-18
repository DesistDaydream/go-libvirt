package virt

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"libvirt.org/go/libvirt"
)

func createVolumes(host *Host, spec CreateSpec, plans []diskPlan) ([]*libvirt.StorageVol, error) {
	pool, err := host.Conn.LookupStoragePoolByName(spec.Pool)
	if err != nil {
		return nil, fmt.Errorf("查找存储池 %s 失败: %w", spec.Pool, err)
	}
	defer pool.Free()

	var volumes []*libvirt.StorageVol
	for _, plan := range plans {
		if existing, err := host.Conn.LookupStorageVolByPath(plan.Path); err == nil {
			_ = existing.Free()
			deleteVolumes(volumes)
			return nil, fmt.Errorf("磁盘已存在: %s", plan.Path)
		}

		var volume *libvirt.StorageVol
		if plan.CopyFrom != "" {
			volume, err = createVolumeFromImage(host, pool, plan)
		} else {
			volume, err = pool.StorageVolCreateXML(volumeXML(plan), libvirt.StorageVolCreateFlags(0))
		}
		if err != nil {
			deleteVolumes(volumes)
			return nil, fmt.Errorf("创建磁盘 %s 失败: %w", plan.Path, err)
		}
		volumes = append(volumes, volume)
	}
	return volumes, nil
}

// createVolumeFromImage 把已有镜像完整拷贝成目标 qcow2，用于 domain_image。
func createVolumeFromImage(host *Host, targetPool *libvirt.StoragePool, plan diskPlan) (*libvirt.StorageVol, error) {
	source, cleanup, err := openImageVolume(host, plan.CopyFrom)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	info, err := source.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("读取镜像 %s 信息失败: %w", plan.CopyFrom, err)
	}
	capacity := plan.Capacity
	if info.Capacity > capacity {
		capacity = info.Capacity
	}

	target := diskPlan{Path: plan.Path, Capacity: capacity}
	volume, err := targetPool.StorageVolCreateXMLFrom(volumeXML(target), source, libvirt.StorageVolCreateFlags(0))
	if err != nil {
		if partial, lookupErr := host.Conn.LookupStorageVolByPath(plan.Path); lookupErr == nil {
			_ = partial.Delete(libvirt.StorageVolDeleteFlags(0))
			_ = partial.Free()
		}
		return nil, fmt.Errorf("拷贝镜像 %s 失败: %w", plan.CopyFrom, err)
	}
	return volume, nil
}

// openImageVolume 打开任意路径的镜像。路径在存储池内时直接查找；
// 否则临时创建一个 dir 存储池把父目录暴露出来，用完销毁。
func openImageVolume(host *Host, path string) (*libvirt.StorageVol, func(), error) {
	if volume, err := host.Conn.LookupStorageVolByPath(path); err == nil {
		return volume, func() { _ = volume.Free() }, nil
	}

	dir := filepath.Dir(path)
	name := filepath.Base(path)
	poolName := fmt.Sprintf("vm-control-src-%d", time.Now().UnixNano())
	poolXML := fmt.Sprintf("<pool type='dir'><name>%s</name><target><path>%s</path></target></pool>",
		escapeXML(poolName), escapeXML(dir))

	pool, err := host.Conn.StoragePoolCreateXML(poolXML, libvirt.StoragePoolCreateFlags(0))
	if err != nil {
		return nil, nil, fmt.Errorf("创建临时存储池失败: %w", err)
	}
	cleanupPool := func() {
		_ = pool.Destroy()
		_ = pool.Free()
	}

	if err := pool.Refresh(0); err != nil {
		cleanupPool()
		return nil, nil, fmt.Errorf("刷新临时存储池失败: %w", err)
	}

	volume, err := pool.LookupStorageVolByName(name)
	if err != nil {
		cleanupPool()
		return nil, nil, fmt.Errorf("查找镜像 %s 失败: %w", path, err)
	}

	return volume, func() {
		_ = volume.Free()
		cleanupPool()
	}, nil
}

func volumeXML(plan diskPlan) string {
	var b strings.Builder
	b.WriteString("<volume>")
	fmt.Fprintf(&b, "<name>%s</name>", escapeXML(filepath.Base(plan.Path)))
	fmt.Fprintf(&b, "<capacity>%d</capacity>", plan.Capacity)
	b.WriteString("<allocation>0</allocation>")
	b.WriteString("<target>")
	fmt.Fprintf(&b, "<path>%s</path>", escapeXML(plan.Path))
	b.WriteString("<format type='qcow2'/>")
	b.WriteString("</target>")
	if plan.Backing != "" {
		b.WriteString("<backingStore>")
		fmt.Fprintf(&b, "<path>%s</path>", escapeXML(plan.Backing))
		b.WriteString("<format type='qcow2'/>")
		b.WriteString("</backingStore>")
	}
	b.WriteString("</volume>")
	return b.String()
}

func escapeXML(value string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(value))
	return buf.String()
}
