package cloudinit

import (
	"strings"
	"testing"
)

func TestBuildISO(t *testing.T) {
	config := Config{
		Name:    "seed-vm",
		NICName: "enp1s0",
		MAC:     "52:54:00:12:34:56",
		Address: "10.10.22.180/23",
		Gateway: "10.10.22.1",
		DNS:     []string{"10.10.22.1"},
	}

	iso, err := BuildISO(config)
	if err != nil {
		t.Fatalf("BuildISO 返回错误: %v", err)
	}
	if len(iso) < 17*2048 {
		t.Fatalf("seed ISO 太小: %d", len(iso))
	}
	pvd := iso[16*2048:]
	if pvd[0] != 1 || string(pvd[1:6]) != "CD001" {
		t.Fatalf("seed ISO 缺少 PVD")
	}
	if label := strings.TrimRight(string(pvd[40:72]), "\x00 "); label != "CIDATA" {
		t.Errorf("volume label = %q，期望 CIDATA", label)
	}
	for _, want := range []string{"meta-data", "user-data", "network-config", "macaddress", "10.10.22.180/23"} {
		if !strings.Contains(string(iso), want) {
			t.Errorf("seed ISO 缺少 %q", want)
		}
	}
}
