//go:build integration

package virt

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"libvirt.org/go/libvirt"
)

func TestCreateCloudInitVolumeIntegration(t *testing.T) {
	uri := os.Getenv("VM_CONTROL_TEST_URI")
	if uri == "" {
		t.Skip("VM_CONTROL_TEST_URI 未设置")
	}

	conn, err := libvirt.NewConnect(uri)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	host := &Host{Conn: conn}
	name := fmt.Sprintf("vm-control-seed-test-%d", time.Now().UnixNano())
	data := []byte("seed-test")

	volume, path, err := createCloudInitVolume(host, "default", name, data)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = volume.Delete(libvirt.StorageVolDeleteFlags(0))
		_ = volume.Free()
	}()

	if path == "" {
		t.Fatal("seed 卷路径为空")
	}
	info, err := volume.GetInfo()
	if err != nil {
		t.Fatal(err)
	}
	if info.Capacity < uint64(len(data)) {
		t.Fatalf("容量 = %d，期望至少 %d", info.Capacity, len(data))
	}

	stream, err := conn.NewStream(0)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Free()
	if err := volume.Download(stream, 0, uint64(len(data)), 0); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(data))
	n, err := stream.Recv(got)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(data) || !bytes.Equal(got, data) {
		t.Fatalf("下载内容 = %q（%d 字节），期望 %q", got, n, data)
	}
	if err := stream.Finish(); err != nil {
		t.Fatal(err)
	}
}
