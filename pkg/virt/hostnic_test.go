package virt

import (
	"testing"
)

func TestParseHostNIC(t *testing.T) {
	xmlDoc := `<device>
  <name>net_enp125s0f0_c4_16_c8_66_be_71</name>
  <parent>pci_0000_7d_00_0</parent>
  <capability type='net'>
    <interface>enp125s0f0</interface>
    <address>c4:16:c8:66:be:71</address>
  </capability>
</device>`

	nic, ok := parseHostNIC(xmlDoc)
	if !ok {
		t.Fatal("parseHostNIC 未识别物理网卡")
	}
	if nic.Name != "enp125s0f0" {
		t.Errorf("name = %q", nic.Name)
	}
	if nic.PCI != "0000:7d:00.0" {
		t.Errorf("pci = %q，期望 0000:7d:00.0", nic.PCI)
	}
}

func TestParseHostNICVirtual(t *testing.T) {
	xmlDoc := `<device>
  <name>net_vnet0_fe_54_00_3b_9a_29</name>
  <parent>computer</parent>
  <capability type='net'>
    <interface>vnet0</interface>
  </capability>
</device>`

	if _, ok := parseHostNIC(xmlDoc); ok {
		t.Error("虚拟网卡不应被识别为可直通物理网卡")
	}
}
