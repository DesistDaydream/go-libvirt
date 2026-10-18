package virt

import "testing"

func TestPCINodeName(t *testing.T) {
	address, err := parsePCIAddress("0000:bd:00.0")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := pciNodeName(address), "pci_0000_bd_00_0"; got != want {
		t.Errorf("pciNodeName = %q，期望 %q", got, want)
	}
}
