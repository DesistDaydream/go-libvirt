package virt

import (
	"crypto/rand"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/beevik/etree"
)

type diskPlan struct {
	Path     string
	Capacity uint64
	Backing  string
	CopyFrom string
}

type createPlan struct {
	XML           string
	Disks         []diskPlan
	MAC           string
	NetworkName   string
	MgmtNICBridge string
}

func buildCreatePlan(spec CreateSpec) (*createPlan, error) {
	if spec.RootVolume == 0 {
		return nil, errors.New("root_volume 不能为 0")
	}
	sources := 0
	for _, source := range []string{spec.DomainImage, spec.BaseImage, spec.ISO} {
		if source != "" {
			sources++
		}
	}
	if sources != 1 {
		return nil, errors.New("domain_image、base_image、iso 必须且只能指定一个")
	}
	if spec.Pool == "" {
		spec.Pool = "default"
	}
	if spec.MaxVCPU == 0 {
		spec.MaxVCPU = spec.VCPU
	}
	if spec.MaxMemory == 0 {
		spec.MaxMemory = spec.Memory
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromFile(spec.XMLPath); err != nil {
		return nil, fmt.Errorf("读取基础 XML 失败: %w", err)
	}
	removeComments(doc)

	root := doc.Root()
	if root == nil || root.Tag != "domain" {
		return nil, errors.New("基础 XML 的根元素不是 <domain>")
	}

	nameElement := root.FindElement("name")
	if nameElement == nil {
		return nil, errors.New("基础 XML 缺少 <name>")
	}
	baseName := strings.TrimSpace(nameElement.Text())
	if baseName == "" {
		return nil, errors.New("基础 XML 的 <name> 为空")
	}

	mac, err := generateMAC()
	if err != nil {
		return nil, err
	}

	nameElement.SetText(spec.Name)
	if err := setMemory(root, spec); err != nil {
		return nil, err
	}
	if err := setVCPU(root, spec); err != nil {
		return nil, err
	}
	setNVRAM(root, baseName, spec.Name)
	removeUUID(root)

	disks, err := planDisks(root, baseName, spec)
	if err != nil {
		return nil, err
	}

	networkName, err := setInterface(root, spec, mac)
	if err != nil {
		return nil, err
	}
	if err := addMacvtapPassthroughNICs(root, spec.MacvtapPassthroughNICs); err != nil {
		return nil, err
	}
	if err := ensureMemballoon(root); err != nil {
		return nil, err
	}
	if err := configureInstallISO(root, spec); err != nil {
		return nil, err
	}
	if err := addPCIPassthroughNICs(root, spec.PCIPassthroughNICs); err != nil {
		return nil, err
	}

	doc.Indent(2)
	body, err := doc.WriteToBytes()
	if err != nil {
		return nil, fmt.Errorf("生成虚拟机 XML 失败: %w", err)
	}

	return &createPlan{
		XML:           string(body),
		Disks:         disks,
		MAC:           mac,
		NetworkName:   networkName,
		MgmtNICBridge: spec.MgmtNICBridge,
	}, nil
}

func setMemory(root *etree.Element, spec CreateSpec) error {
	if spec.Memory < 1024 || spec.MaxMemory < 1024 {
		return errors.New("内存至少为 1KiB")
	}

	maxKiB := spec.MaxMemory / 1024
	currentKiB := spec.Memory / 1024

	memory := root.FindElement("memory")
	if memory == nil {
		return errors.New("基础 XML 缺少 <memory>")
	}
	memory.CreateAttr("unit", "KiB")
	memory.SetText(strconv.FormatUint(maxKiB, 10))

	current := root.FindElement("currentMemory")
	if current == nil {
		return errors.New("基础 XML 缺少 <currentMemory>")
	}
	current.CreateAttr("unit", "KiB")
	current.SetText(strconv.FormatUint(currentKiB, 10))
	return nil
}

func setVCPU(root *etree.Element, spec CreateSpec) error {
	if spec.VCPU <= 0 || spec.MaxVCPU < spec.VCPU {
		return errors.New("CPU 配置不合法")
	}

	vcpu := root.FindElement("vcpu")
	if vcpu == nil {
		return errors.New("基础 XML 缺少 <vcpu>")
	}
	vcpu.CreateAttr("placement", "static")
	vcpu.SetText(strconv.Itoa(spec.MaxVCPU))
	if spec.VCPU < spec.MaxVCPU {
		vcpu.CreateAttr("current", strconv.Itoa(spec.VCPU))
	} else {
		vcpu.RemoveAttr("current")
	}
	return nil
}

func setNVRAM(root *etree.Element, baseName, name string) {
	nvram := root.FindElement("os/nvram")
	if nvram == nil {
		return
	}
	old := nvram.Text()
	if old == "" {
		return
	}
	if strings.Contains(old, baseName) {
		nvram.SetText(strings.ReplaceAll(old, baseName, name))
		return
	}
	if strings.EqualFold(filepath.Ext(old), ".fd") {
		nvram.SetText(filepath.Join(filepath.Dir(old), name+"_VARS.fd"))
	}
}

func removeUUID(root *etree.Element) {
	if uuid := root.FindElement("uuid"); uuid != nil {
		root.RemoveChild(uuid)
	}
}

func removeComments(doc *etree.Document) {
	for i := len(doc.Child) - 1; i >= 0; i-- {
		if _, ok := doc.Child[i].(*etree.Comment); ok {
			doc.RemoveChild(doc.Child[i])
		}
	}
}

func planDisks(root *etree.Element, baseName string, spec CreateSpec) ([]diskPlan, error) {
	var plans []diskPlan
	index := 0

	for _, disk := range root.FindElements("devices/disk") {
		if disk.SelectAttrValue("device", "") != "disk" {
			continue
		}
		source := disk.FindElement("source")
		if source == nil {
			continue
		}
		oldPath := source.SelectAttrValue("file", "")
		if oldPath == "" {
			continue
		}

		newPath := deriveDiskPath(oldPath, baseName, spec.Name, index)
		if newPath == oldPath {
			return nil, fmt.Errorf("磁盘路径未变化: %s", oldPath)
		}
		source.CreateAttr("file", newPath)

		plan := diskPlan{Path: newPath, Capacity: spec.RootVolume}
		if index == 0 {
			if spec.DomainImage != "" {
				plan.CopyFrom = spec.DomainImage
			} else {
				plan.Backing = spec.BaseImage
			}
		}
		plans = append(plans, plan)
		index++
	}

	if len(plans) == 0 {
		return nil, errors.New("基础 XML 中没有 device=disk 的磁盘")
	}

	baseDir := filepath.Dir(plans[0].Path)
	usedDevices := make(map[string]bool)
	for _, disk := range root.FindElements("devices/disk") {
		if target := disk.FindElement("target"); target != nil {
			usedDevices[target.SelectAttrValue("dev", "")] = true
		}
	}
	usedNames := make(map[string]bool, len(plans)+len(spec.Volumes))
	for _, plan := range plans {
		usedNames[filepath.Base(plan.Path)] = true
	}

	for _, volume := range spec.Volumes {
		mount := strings.TrimSpace(volume.Path)
		if mount == "" {
			return nil, errors.New("volumes 中的挂载路径不能为空")
		}
		name, err := volumeName(mount)
		if err != nil {
			return nil, err
		}
		filename := fmt.Sprintf("%s-%s.qcow2", spec.Name, name)
		if usedNames[filename] {
			return nil, fmt.Errorf("volume 文件名重复: %s", filename)
		}
		usedNames[filename] = true

		path := filepath.Join(baseDir, filename)
		dev, err := nextTargetDev(usedDevices)
		if err != nil {
			return nil, err
		}
		if err := addVolumeDisk(root, path, dev); err != nil {
			return nil, err
		}
		capacity := volume.Size
		if capacity == 0 {
			capacity = spec.RootVolume
		}
		plans = append(plans, diskPlan{Path: path, Capacity: capacity})
	}

	return plans, nil
}

var unsafeVolumeChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func volumeName(mount string) (string, error) {
	trimmed := strings.Trim(mount, "/")
	if trimmed == "" {
		return "", fmt.Errorf("volume 挂载路径不能为 /: %q", mount)
	}
	name := unsafeVolumeChars.ReplaceAllString(trimmed, "-")
	name = strings.Trim(name, "-")
	if name == "" {
		return "", fmt.Errorf("volume 挂载路径无法生成文件名: %q", mount)
	}
	return name, nil
}

func nextTargetDev(used map[string]bool) (string, error) {
	for i := 1; i < 26; i++ {
		dev := fmt.Sprintf("vd%c", 'a'+rune(i))
		if !used[dev] {
			used[dev] = true
			return dev, nil
		}
	}
	return "", errors.New("virtio 磁盘槽位已用完")
}

func addVolumeDisk(root *etree.Element, path, dev string) error {
	devices := root.FindElement("devices")
	if devices == nil {
		return errors.New("基础 XML 缺少 <devices>")
	}

	disk := etree.NewElement("disk")
	disk.CreateAttr("type", "file")
	disk.CreateAttr("device", "disk")

	driver := disk.CreateElement("driver")
	driver.CreateAttr("name", "qemu")
	driver.CreateAttr("type", "qcow2")

	source := disk.CreateElement("source")
	source.CreateAttr("file", path)

	target := disk.CreateElement("target")
	target.CreateAttr("dev", dev)
	target.CreateAttr("bus", "virtio")

	devices.AddChild(disk)
	return nil
}

// configureInstallISO 在没有 base_image 时挂载安装 ISO，并把光盘设为第一启动项。
func configureInstallISO(root *etree.Element, spec CreateSpec) error {
	if spec.ISO == "" {
		return nil
	}

	devices := root.FindElement("devices")
	if devices == nil {
		return errors.New("基础 XML 缺少 <devices>")
	}

	cdrom := findDiskByDevice(root, "cdrom")
	if cdrom == nil {
		cdrom = etree.NewElement("disk")
		cdrom.CreateAttr("type", "file")
		cdrom.CreateAttr("device", "cdrom")

		driver := cdrom.CreateElement("driver")
		driver.CreateAttr("name", "qemu")
		driver.CreateAttr("type", "raw")

		source := cdrom.CreateElement("source")
		source.CreateAttr("file", spec.ISO)

		dev, err := nextSCSITargetDev(root)
		if err != nil {
			return err
		}
		target := cdrom.CreateElement("target")
		target.CreateAttr("dev", dev)
		target.CreateAttr("bus", "scsi")
		cdrom.CreateElement("readonly")

		devices.AddChild(cdrom)
	} else {
		source := cdrom.FindElement("source")
		if source == nil {
			source = cdrom.CreateElement("source")
		}
		source.CreateAttr("file", spec.ISO)
		if cdrom.FindElement("readonly") == nil {
			cdrom.CreateElement("readonly")
		}
	}

	removeDiskBootOrders(root)
	return setInstallBootOrder(root)
}

func findDiskByDevice(root *etree.Element, device string) *etree.Element {
	for _, disk := range root.FindElements("devices/disk") {
		if disk.SelectAttrValue("device", "") == device {
			return disk
		}
	}
	return nil
}

func nextSCSITargetDev(root *etree.Element) (string, error) {
	used := make(map[string]bool)
	for _, disk := range root.FindElements("devices/disk") {
		if target := disk.FindElement("target"); target != nil {
			used[target.SelectAttrValue("dev", "")] = true
		}
	}
	for i := 0; i < 26; i++ {
		dev := fmt.Sprintf("sd%c", 'a'+rune(i))
		if !used[dev] {
			return dev, nil
		}
	}
	return "", errors.New("scsi 磁盘槽位已用完")
}

func removeDiskBootOrders(root *etree.Element) {
	for _, disk := range root.FindElements("devices/disk") {
		for _, boot := range disk.FindElements("boot") {
			disk.RemoveChild(boot)
		}
	}
}

func setInstallBootOrder(root *etree.Element) error {
	osElement := root.FindElement("os")
	if osElement == nil {
		return errors.New("基础 XML 缺少 <os>")
	}
	for _, boot := range osElement.FindElements("boot") {
		osElement.RemoveChild(boot)
	}
	osElement.CreateElement("boot").CreateAttr("dev", "cdrom")
	osElement.CreateElement("boot").CreateAttr("dev", "hd")
	return nil
}

// addPCIPassthroughNICs 把宿主机上的多块 PCI 网卡直通给虚拟机。
func addPCIPassthroughNICs(root *etree.Element, pcis []string) error {
	if len(pcis) == 0 {
		return nil
	}

	devices := root.FindElement("devices")
	if devices == nil {
		return errors.New("基础 XML 缺少 <devices>")
	}

	seen := make(map[string]bool, len(pcis))
	for _, pci := range pcis {
		pci = strings.TrimSpace(pci)
		if pci == "" {
			return errors.New("pci_passthrough_nics 中的 PCI 地址不能为空")
		}
		if seen[pci] {
			return fmt.Errorf("passthrough NIC 重复: %s", pci)
		}
		seen[pci] = true

		address, err := parsePCIAddress(pci)
		if err != nil {
			return err
		}

		hostdev := etree.NewElement("hostdev")
		hostdev.CreateAttr("mode", "subsystem")
		hostdev.CreateAttr("type", "pci")
		hostdev.CreateAttr("managed", "yes")

		source := hostdev.CreateElement("source")
		addr := source.CreateElement("address")
		addr.CreateAttr("domain", fmt.Sprintf("0x%04x", address.Domain))
		addr.CreateAttr("bus", fmt.Sprintf("0x%02x", address.Bus))
		addr.CreateAttr("slot", fmt.Sprintf("0x%02x", address.Slot))
		addr.CreateAttr("function", fmt.Sprintf("0x%x", address.Function))

		devices.AddChild(hostdev)
	}
	return nil
}

type pciAddress struct {
	Domain   uint32
	Bus      uint32
	Slot     uint32
	Function uint32
}

var pciAddressPattern = regexp.MustCompile(`^(?:([0-9a-fA-F]{4}):)?([0-9a-fA-F]{2}):([0-9a-fA-F]{2})\.([0-7])$`)

func parsePCIAddress(value string) (pciAddress, error) {
	match := pciAddressPattern.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return pciAddress{}, fmt.Errorf("无效的 PCI 地址 %q（示例：0000:3b:00.0）", value)
	}

	parse := func(s string) uint32 {
		n, _ := strconv.ParseUint(s, 16, 32)
		return uint32(n)
	}

	var domain uint32
	if match[1] != "" {
		domain = parse(match[1])
	}
	return pciAddress{
		Domain:   domain,
		Bus:      parse(match[2]),
		Slot:     parse(match[3]),
		Function: parse(match[4]),
	}, nil
}

func deriveDiskPath(oldPath, baseName, name string, index int) string {
	base := filepath.Base(oldPath)
	newBase := base
	if strings.Contains(base, baseName) {
		newBase = strings.Replace(base, baseName, name, 1)
	}
	if newBase == base {
		if index == 0 {
			newBase = name + ".qcow2"
		} else {
			newBase = fmt.Sprintf("%s-%d.qcow2", name, index)
		}
	}
	return filepath.Join(filepath.Dir(oldPath), newBase)
}

func setInterface(root *etree.Element, spec CreateSpec, mac string) (string, error) {
	devices := root.FindElement("devices")
	if devices == nil {
		return "", errors.New("基础 XML 缺少 <devices>")
	}

	interfaces := root.FindElements("devices/interface")
	networkName := ""
	if len(interfaces) > 0 {
		if source := interfaces[0].FindElement("source"); source != nil {
			networkName = source.SelectAttrValue("network", "")
		}
	}

	if spec.MgmtNICBridge == "" {
		if len(interfaces) == 0 {
			return "", errors.New("基础 XML 没有网络接口，且未指定 mgmt_nic_bridge")
		}
		setMAC(interfaces[0], mac)
		return networkName, nil
	}

	for _, iface := range interfaces {
		devices.RemoveChild(iface)
	}
	iface := etree.NewElement("interface")
	iface.CreateAttr("type", "bridge")
	source := iface.CreateElement("source")
	source.CreateAttr("bridge", spec.MgmtNICBridge)
	model := iface.CreateElement("model")
	model.CreateAttr("type", "virtio")
	macElement := iface.CreateElement("mac")
	macElement.CreateAttr("address", mac)
	devices.AddChild(iface)

	return networkName, nil
}

func setMAC(iface *etree.Element, mac string) {
	if element := iface.FindElement("mac"); element != nil {
		element.CreateAttr("address", mac)
		return
	}
	element := iface.CreateElement("mac")
	element.CreateAttr("address", mac)
}

// addMacvtapPassthroughNICs 为每个宿主机镜像口增加一块 virtio 网卡，macvtap passthrough 模式。
func addMacvtapPassthroughNICs(root *etree.Element, devs []string) error {
	if len(devs) == 0 {
		return nil
	}

	devices := root.FindElement("devices")
	if devices == nil {
		return errors.New("基础 XML 缺少 <devices>")
	}

	for _, dev := range devs {
		iface := etree.NewElement("interface")
		iface.CreateAttr("type", "direct")

		source := iface.CreateElement("source")
		source.CreateAttr("dev", dev)
		source.CreateAttr("mode", "passthrough")

		model := iface.CreateElement("model")
		model.CreateAttr("type", "virtio")

		devices.AddChild(iface)
	}
	return nil
}

// ensureMemballoon 保证存在 virtio-balloon，否则 currentMemory 无法低于 memory。
func ensureMemballoon(root *etree.Element) error {
	devices := root.FindElement("devices")
	if devices == nil {
		return errors.New("基础 XML 缺少 <devices>")
	}
	if devices.FindElement("memballoon") != nil {
		return nil
	}

	balloon := etree.NewElement("memballoon")
	balloon.CreateAttr("model", "virtio")
	devices.AddChild(balloon)
	return nil
}

func generateMAC() (string, error) {
	var buf [3]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("生成 MAC 失败: %w", err)
	}
	return fmt.Sprintf("52:54:00:%02x:%02x:%02x", buf[0], buf[1], buf[2]), nil
}
