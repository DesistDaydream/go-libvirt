package cloudinit

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	config, err := New(Config{
		Name:    "seed-vm",
		MAC:     "52:54:00:12:34:56",
		Address: "10.10.22.180/23",
		Gateway: "10.10.22.1",
	})
	if err != nil {
		t.Fatalf("New 返回错误: %v", err)
	}
	if config.NICName != "enp1s0" {
		t.Errorf("NICName = %q，期望默认 enp1s0", config.NICName)
	}

	network := config.NetworkConfig()
	for _, want := range []string{"enp1s0:", "set-name: enp1s0", "10.10.22.180/23", "to: 0.0.0.0/0", "dhcp4: false"} {
		if !strings.Contains(network, want) {
			t.Errorf("network-config 缺少 %q:\n%s", want, network)
		}
	}

	config, err = New(Config{
		Name:    "seed-vm",
		NICName: "ens1",
		MAC:     "52:54:00:12:34:56",
		Address: "10.10.22.180/23",
		Gateway: "10.10.22.1",
	})
	if err != nil {
		t.Fatalf("New 返回错误: %v", err)
	}
	if config.NICName != "ens1" {
		t.Errorf("NICName = %q，期望 ens1", config.NICName)
	}

	if _, err := New(Config{
		Name:    "seed-vm",
		NICName: "bad/name",
		MAC:     "52:54:00:12:34:56",
		Address: "10.10.22.180/23",
		Gateway: "10.10.22.1",
	}); err == nil {
		t.Fatal("非法网卡名应返回错误")
	}
}
