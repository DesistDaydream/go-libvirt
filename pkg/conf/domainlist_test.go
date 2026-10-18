package conf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseByteSize(t *testing.T) {
	tests := []struct {
		input   string
		want    uint64
		wantErr bool
	}{
		{"4GiB", 4 << 30, false},
		{"20G", 20 << 30, false},
		{"1024MiB", 1 << 30, false},
		{"1.5GiB", 3 << 29, false},
		{"512", 512, false},
		{"abc", 0, true},
		{"0", 0, true},
	}

	for _, test := range tests {
		got, err := parseByteSize(test.input)
		if test.wantErr {
			if err == nil {
				t.Errorf("parseByteSize(%q) 期望错误，实际没有", test.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseByteSize(%q) 返回错误: %v", test.input, err)
			continue
		}
		if got != test.want {
			t.Errorf("parseByteSize(%q) = %d，期望 %d", test.input, got, test.want)
		}
	}
}

func TestLoadDomainList(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "base.xml")
	if err := os.WriteFile(xmlPath, []byte("<domain/>"), 0644); err != nil {
		t.Fatal(err)
	}

	listPath := filepath.Join(dir, "domain-list.yaml")
	content := `domains:
  - name: test-01
    vcpu: 2
    memory: 2GiB
    root_volume: 10GiB
    volumes:
      - /data
      - path: /var/log
        size: 5GiB
    mgmt_nic_bridge: br0
    macvtap_passthrough_nics:
      - enp189s0f0
      - enp189s0f1
    xml: base.xml
    base_image: /base.qcow2
`
	if err := os.WriteFile(listPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	list, err := LoadDomainList(listPath)
	if err != nil {
		t.Fatalf("LoadDomainList 返回错误: %v", err)
	}
	if len(list.Domains) != 1 {
		t.Fatalf("domains 数量 = %d，期望 1", len(list.Domains))
	}

	spec := list.Domains[0]
	if spec.XML != xmlPath {
		t.Errorf("xml = %q，期望 %q", spec.XML, xmlPath)
	}
	if spec.MaxVCPU != 2 {
		t.Errorf("max_vcpu = %d，期望默认 2", spec.MaxVCPU)
	}
	if spec.MaxMemory != 2<<30 {
		t.Errorf("max_memory = %d，期望默认 2GiB", spec.MaxMemory)
	}
	if spec.RootVolume != 10<<30 {
		t.Errorf("root_volume = %d，期望 10GiB", spec.RootVolume)
	}
	if len(spec.Volumes) != 2 {
		t.Fatalf("volumes 数量 = %d，期望 2", len(spec.Volumes))
	}
	if spec.Volumes[0].Path != "/data" || spec.Volumes[0].Size != 0 {
		t.Errorf("第一个 volume = %+v，期望 /data 且未指定 size", spec.Volumes[0])
	}
	if spec.Volumes[1].Size != 5<<30 {
		t.Errorf("第二个 volume size = %d，期望 5GiB", spec.Volumes[1].Size)
	}
	if spec.BaseImage != "/base.qcow2" {
		t.Errorf("base_image = %q", spec.BaseImage)
	}
	if len(spec.MacvtapPassthroughNICs) != 2 || spec.MacvtapPassthroughNICs[0] != "enp189s0f0" || spec.MacvtapPassthroughNICs[1] != "enp189s0f1" {
		t.Errorf("macvtap_passthrough_nics = %v，期望 [enp189s0f0 enp189s0f1]", spec.MacvtapPassthroughNICs)
	}

	createSpec := spec.ToCreateSpec()
	if createSpec.Volumes[0].Size != 10<<30 {
		t.Errorf("默认 volume size = %d，期望继承 root_volume", createSpec.Volumes[0].Size)
	}
	if spec.Pool != "default" {
		t.Errorf("pool = %q，期望 default", spec.Pool)
	}
}

func TestLoadDomainListRequiresOneImageSource(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "base.xml")
	if err := os.WriteFile(xmlPath, []byte("<domain/>"), 0644); err != nil {
		t.Fatal(err)
	}

	listPath := filepath.Join(dir, "domain-list.yaml")
	content := `domains:
  - name: test-01
    vcpu: 2
    memory: 2GiB
    root_volume: 10GiB
    mgmt_nic_bridge: br0
    xml: base.xml
    domain_image: /domain.qcow2
    base_image: /base.qcow2
`
	if err := os.WriteFile(listPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadDomainList(listPath); err == nil {
		t.Fatal("同时指定 domain_image 和 base_image 时应返回错误")
	}
}

func TestLoadDomainListMirrorAndPassthrough(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "base.xml")
	if err := os.WriteFile(xmlPath, []byte("<domain/>"), 0644); err != nil {
		t.Fatal(err)
	}

	listPath := filepath.Join(dir, "domain-list.yaml")
	content := `domains:
  - name: test-01
    vcpu: 2
    memory: 2GiB
    root_volume: 10GiB
    mgmt_nic_bridge: br0
    xml: base.xml
    macvtap_passthrough_nics:
      - enp189s0f0
    pci_passthrough_nics:
      - "0000:bd:00.0"
    domain_image: /domain.qcow2
`
	if err := os.WriteFile(listPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadDomainList(listPath); err == nil {
		t.Fatal("同时指定 macvtap_passthrough_nics 和 pci_passthrough_nics 时应返回错误")
	}
}
