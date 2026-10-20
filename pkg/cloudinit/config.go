// Package cloudinit 生成 cloud-init NoCloud seed 的配置和 ISO。
package cloudinit

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Config 是一台 VM 的 cloud-init seed 配置。
type Config struct {
	Name    string
	NICName string
	MAC     string
	Address string // CIDR
	Gateway string
	DNS     []string
}

// New 校验并规范化 seed 配置。
func New(config Config) (*Config, error) {
	if strings.TrimSpace(config.Name) == "" {
		return nil, errors.New("实例名不能为空")
	}
	if config.MAC == "" {
		return nil, errors.New("MAC 不能为空")
	}
	if config.Address == "" {
		return nil, errors.New("ip 不能为空")
	}
	if config.Gateway == "" {
		return nil, errors.New("配置 ip 时必须指定 gateway")
	}

	ip, network, err := net.ParseCIDR(config.Address)
	if err != nil || ip.To4() == nil {
		return nil, fmt.Errorf("ip %q 不是 IPv4 CIDR", config.Address)
	}

	gateway := net.ParseIP(config.Gateway)
	if gateway == nil || gateway.To4() == nil {
		return nil, fmt.Errorf("gateway %q 不是 IPv4 地址", config.Gateway)
	}
	if !network.Contains(gateway) {
		return nil, fmt.Errorf("gateway %q 不在 ip %q 的网段内", config.Gateway, config.Address)
	}

	for _, dns := range config.DNS {
		if value := net.ParseIP(dns); value == nil || value.To4() == nil {
			return nil, fmt.Errorf("dns %q 不是 IPv4 地址", dns)
		}
	}

	nicName := strings.TrimSpace(config.NICName)
	if nicName == "" {
		nicName = "enp1s0"
	}
	if strings.ContainsAny(nicName, " \t/") {
		return nil, fmt.Errorf("mgmt_nic_name %q 不是合法的网卡名", config.NICName)
	}
	config.NICName = nicName

	return &config, nil
}

// MetaData 生成 NoCloud 的 meta-data。
func (c Config) MetaData() string {
	return fmt.Sprintf("instance-id: %s\nlocal-hostname: %s\n",
		yamlQuote(c.Name), yamlQuote(c.Name))
}

// UserData 生成最小 cloud-config。
func (c Config) UserData() string {
	return fmt.Sprintf("#cloud-config\nhostname: %s\n", yamlQuote(c.Name))
}

// NetworkConfig 生成 NetworkManager 可用的 network-config v2。
func (c Config) NetworkConfig() string {
	nicName := strings.TrimSpace(c.NICName)
	if nicName == "" {
		nicName = "enp1s0"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "version: 2\nethernets:\n  %s:\n    match:\n      macaddress: %s\n"+
		"    set-name: %s\n    dhcp4: false\n    addresses:\n      - %s\n"+
		"    routes:\n      - to: 0.0.0.0/0\n        via: %s\n",
		nicName, yamlQuote(c.MAC), nicName, c.Address, c.Gateway)

	if len(c.DNS) > 0 {
		b.WriteString("    nameservers:\n      addresses:\n")
		for _, dns := range c.DNS {
			fmt.Fprintf(&b, "        - %s\n", dns)
		}
	}
	return b.String()
}

func yamlQuote(value string) string {
	return strconv.Quote(value)
}
