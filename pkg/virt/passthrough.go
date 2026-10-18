package virt

import (
	"fmt"
	"strings"
)

// validatePCIPassthroughNICs 在创建磁盘前确认宿主机能为每个 PCI 设备提供 IOMMU group。
func validatePCIPassthroughNICs(host *Host, pcis []string) error {
	for _, pci := range pcis {
		address, err := parsePCIAddress(pci)
		if err != nil {
			return err
		}
		name := pciNodeName(address)

		device, err := host.Conn.LookupDeviceByName(name)
		if err != nil {
			return fmt.Errorf("查找 PCI 设备 %s 失败: %w", pci, err)
		}
		xmlDesc, xmlErr := device.GetXMLDesc(0)
		_ = device.Free()
		if xmlErr != nil {
			return fmt.Errorf("读取 PCI 设备 %s 信息失败: %w", pci, xmlErr)
		}

		if !strings.Contains(xmlDesc, "<iommuGroup") {
			return fmt.Errorf("宿主机 %s 无法直通 PCI 设备 %s：没有 IOMMU group，请先在 BIOS/UEFI 开启 SMMU/IOMMU 并重启", host.Name, pci)
		}
	}
	return nil
}

func pciNodeName(address pciAddress) string {
	return fmt.Sprintf("pci_%04x_%02x_%02x_%x", address.Domain, address.Bus, address.Slot, address.Function)
}

// validateMacvtapPassthroughNICs 确认 macvtap_passthrough_nics 都指向宿主机上可用且不重复的物理网卡。
func validateMacvtapPassthroughNICs(host *Host, devs []string) error {
	if len(devs) == 0 {
		return nil
	}
	nics, err := HostNICs(host)
	if err != nil {
		return err
	}

	available := make(map[string]bool, len(nics))
	for _, nic := range nics {
		available[nic.Name] = true
	}

	seen := make(map[string]bool, len(devs))
	for _, dev := range devs {
		if !available[dev] {
			return fmt.Errorf("宿主机 %s 上找不到网卡 %s", host.Name, dev)
		}
		if seen[dev] {
			return fmt.Errorf("macvtap_passthrough_nics 中重复指定网卡 %s", dev)
		}
		seen[dev] = true
	}
	return nil
}
