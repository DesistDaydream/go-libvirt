package virt

import (
	"encoding/xml"
	"fmt"
	"sort"
	"strings"

	"libvirt.org/go/libvirt"
)

type HostNIC struct {
	Name string
	PCI  string
}

// HostNICs 列出宿主机上可直通的物理网卡及其 PCI 地址。
func HostNICs(host *Host) ([]HostNIC, error) {
	devices, err := host.Conn.ListAllNodeDevices(libvirt.CONNECT_LIST_NODE_DEVICES_CAP_NET)
	if err != nil {
		return nil, fmt.Errorf("列出 %s 的网卡失败: %w", host.Name, err)
	}
	defer func() {
		for i := range devices {
			_ = devices[i].Free()
		}
	}()

	var nics []HostNIC
	for i := range devices {
		xmlDesc, err := devices[i].GetXMLDesc(0)
		if err != nil {
			continue
		}
		if nic, ok := parseHostNIC(xmlDesc); ok {
			nics = append(nics, nic)
		}
	}

	sort.Slice(nics, func(i, j int) bool {
		return nics[i].Name < nics[j].Name
	})
	return nics, nil
}

type nodeNICXML struct {
	Parent     string `xml:"parent"`
	Capability struct {
		Interface string `xml:"interface"`
	} `xml:"capability"`
}

func parseHostNIC(xmlDoc string) (HostNIC, bool) {
	var device nodeNICXML
	if err := xml.Unmarshal([]byte(xmlDoc), &device); err != nil {
		return HostNIC{}, false
	}
	if device.Capability.Interface == "" {
		return HostNIC{}, false
	}
	pci := pciFromNodeName(device.Parent)
	if pci == "" {
		return HostNIC{}, false
	}
	return HostNIC{Name: device.Capability.Interface, PCI: pci}, true
}

// pciFromNodeName 把 pci_0000_7d_00_0 转成 0000:7d:00.0。
func pciFromNodeName(name string) string {
	if !strings.HasPrefix(name, "pci_") {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(name, "pci_"), "_")
	if len(parts) != 4 {
		return ""
	}
	return fmt.Sprintf("%s:%s:%s.%s", parts[0], parts[1], parts[2], parts[3])
}
