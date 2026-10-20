package virt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beevik/etree"
)

const testBaseDomainXML = `<domain type='kvm'>
  <name>base-vm</name>
  <uuid>11111111-2222-3333-4444-555555555555</uuid>
  <memory unit='KiB'>1048576</memory>
  <currentMemory unit='KiB'>1048576</currentMemory>
  <vcpu placement='static'>2</vcpu>
  <os>
    <type arch='aarch64' machine='virt-6.2'>hvm</type>
    <nvram>/var/lib/libvirt/qemu/nvram/base-vm_VARS.fd</nvram>
  </os>
  <devices>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source file='/var/lib/libvirt/images/base-vm.qcow2'/>
      <target dev='vda' bus='virtio'/>
    </disk>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source file='/var/lib/libvirt/images/base-vm-home.qcow2'/>
      <target dev='vdb' bus='virtio'/>
    </disk>
    <interface type='network'>
      <source network='default'/>
      <model type='virtio'/>
    </interface>
  </devices>
</domain>`

func TestBuildCreatePlan(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "base.xml")
	if err := os.WriteFile(xmlPath, []byte(testBaseDomainXML), 0644); err != nil {
		t.Fatal(err)
	}

	spec := CreateSpec{
		Name:               "test-vm",
		VCPU:               4,
		MaxVCPU:            8,
		Memory:             4 << 30,
		MaxMemory:          8 << 30,
		RootVolume:         20 << 30,
		Volumes:            []CreateVolume{{Path: "/data"}, {Path: "/var/log", Size: 5 << 30}},
		IP:                 "192.168.122.200/24",
		Gateway:            "192.168.122.1",
		MgmtNICBridge:      "br0",
		PCIPassthroughNICs: []string{"0000:3b:00.0", "0000:3b:00.1"},
		BaseImage:          "/var/lib/libvirt/images/base.qcow2",
		Pool:               "default",
		XMLPath:            xmlPath,
	}

	plan, err := buildCreatePlan(spec)
	if err != nil {
		t.Fatalf("buildCreatePlan 返回错误: %v", err)
	}

	if len(plan.Disks) != 4 {
		t.Fatalf("磁盘数量 = %d，期望 4", len(plan.Disks))
	}
	if got, want := plan.Disks[0].Path, "/var/lib/libvirt/images/test-vm.qcow2"; got != want {
		t.Errorf("第一块磁盘 = %q，期望 %q", got, want)
	}
	if plan.Disks[0].Backing != spec.BaseImage {
		t.Errorf("第一块磁盘 backing = %q，期望 %q", plan.Disks[0].Backing, spec.BaseImage)
	}
	if got, want := plan.Disks[1].Path, "/var/lib/libvirt/images/test-vm-home.qcow2"; got != want {
		t.Errorf("第二块磁盘 = %q，期望 %q", got, want)
	}
	if plan.Disks[1].Backing != "" {
		t.Errorf("第二块磁盘不应有 backing，实际 %q", plan.Disks[1].Backing)
	}
	if got, want := plan.Disks[2].Path, "/var/lib/libvirt/images/test-vm-data.qcow2"; got != want {
		t.Errorf("数据盘 = %q，期望 %q", got, want)
	}
	if plan.Disks[2].Capacity != 20<<30 {
		t.Errorf("数据盘容量 = %d，期望默认 root_volume", plan.Disks[2].Capacity)
	}
	if got, want := plan.Disks[3].Path, "/var/lib/libvirt/images/test-vm-var-log.qcow2"; got != want {
		t.Errorf("日志盘 = %q，期望 %q", got, want)
	}
	if plan.Disks[3].Capacity != 5<<30 {
		t.Errorf("日志盘容量 = %d，期望 5GiB", plan.Disks[3].Capacity)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromString(plan.XML); err != nil {
		t.Fatalf("生成的 XML 无法解析: %v\n%s", err, plan.XML)
	}
	root := doc.Root()
	if got := root.FindElement("name").Text(); got != "test-vm" {
		t.Errorf("name = %q", got)
	}
	if got := root.FindElement("memory").Text(); got != "8388608" {
		t.Errorf("memory = %q，期望 8388608", got)
	}
	if got := root.FindElement("currentMemory").Text(); got != "4194304" {
		t.Errorf("currentMemory = %q，期望 4194304", got)
	}
	vcpu := root.FindElement("vcpu")
	if got := vcpu.Text(); got != "8" {
		t.Errorf("vcpu = %q，期望 8", got)
	}
	if got := vcpu.SelectAttrValue("current", ""); got != "4" {
		t.Errorf("vcpu current = %q，期望 4", got)
	}
	if got := root.FindElement("os/nvram").Text(); got != "/var/lib/libvirt/qemu/nvram/test-vm_VARS.fd" {
		t.Errorf("nvram = %q", got)
	}
	if root.FindElement("uuid") != nil {
		t.Error("生成的 XML 仍包含 uuid")
	}
	iface := root.FindElement("devices/interface")
	if got := iface.SelectAttrValue("type", ""); got != "bridge" {
		t.Errorf("interface type = %q，期望 bridge", got)
	}
	if got := iface.FindElement("source").SelectAttrValue("bridge", ""); got != "br0" {
		t.Errorf("interface bridge = %q，期望 br0", got)
	}
	if got := iface.FindElement("mac").SelectAttrValue("address", ""); got != plan.MAC {
		t.Errorf("interface mac = %q，期望 %q", got, plan.MAC)
	}
	if root.FindElement("devices/memballoon") == nil {
		t.Error("生成的 XML 缺少 <memballoon>")
	}
	hostdevs := root.FindElements("devices/hostdev")
	if len(hostdevs) != 2 {
		t.Fatalf("直通网卡数量 = %d，期望 2", len(hostdevs))
	}
	address := hostdevs[0].FindElement("source/address")
	if got := address.SelectAttrValue("bus", ""); got != "0x3b" {
		t.Errorf("hostdev bus = %q，期望 0x3b", got)
	}
	if got := address.SelectAttrValue("slot", ""); got != "0x00" {
		t.Errorf("hostdev slot = %q，期望 0x00", got)
	}
	second := hostdevs[1].FindElement("source/address")
	if got := second.SelectAttrValue("function", ""); got != "0x1" {
		t.Errorf("第二块 hostdev function = %q，期望 0x1", got)
	}
	if strings.Contains(plan.XML, "base-vm") {
		t.Error("生成的 XML 仍包含 base-vm")
	}
}

func TestBuildCreatePlanDomainImage(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "base.xml")
	if err := os.WriteFile(xmlPath, []byte(testBaseDomainXML), 0644); err != nil {
		t.Fatal(err)
	}

	spec := CreateSpec{
		Name:        "image-vm",
		VCPU:        1,
		MaxVCPU:     1,
		Memory:      1 << 30,
		MaxMemory:   1 << 30,
		RootVolume:  50 << 30,
		DomainImage: "/var/lib/libvirt/images/domain.qcow2",
		Pool:        "default",
		XMLPath:     xmlPath,
	}

	plan, err := buildCreatePlan(spec)
	if err != nil {
		t.Fatalf("buildCreatePlan 返回错误: %v", err)
	}
	if plan.Disks[0].CopyFrom != spec.DomainImage {
		t.Errorf("根盘 CopyFrom = %q，期望 %q", plan.Disks[0].CopyFrom, spec.DomainImage)
	}
	if plan.Disks[0].Backing != "" {
		t.Errorf("domain_image 模式不应有 backing，实际 %q", plan.Disks[0].Backing)
	}
	if got, want := plan.Disks[0].Path, "/var/lib/libvirt/images/image-vm.qcow2"; got != want {
		t.Errorf("根盘路径 = %q，期望 %q", got, want)
	}
}

func TestBuildCreatePlanMacvtapPassthroughNICs(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "base.xml")
	if err := os.WriteFile(xmlPath, []byte(testBaseDomainXML), 0644); err != nil {
		t.Fatal(err)
	}

	spec := CreateSpec{
		Name:                   "mirror-vm",
		VCPU:                   1,
		MaxVCPU:                1,
		Memory:                 1 << 30,
		MaxMemory:              1 << 30,
		RootVolume:             10 << 30,
		MgmtNICBridge:          "virbr0",
		MacvtapPassthroughNICs: []string{"enp189s0f0", "enp189s0f1"},
		BaseImage:              "/base.qcow2",
		Pool:                   "default",
		XMLPath:                xmlPath,
	}

	plan, err := buildCreatePlan(spec)
	if err != nil {
		t.Fatalf("buildCreatePlan 返回错误: %v", err)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromString(plan.XML); err != nil {
		t.Fatalf("解析生成的 XML 失败: %v", err)
	}
	interfaces := doc.Root().FindElements("devices/interface")
	if len(interfaces) != 3 {
		t.Fatalf("interface 数量 = %d，期望 3", len(interfaces))
	}

	for i, wantDev := range spec.MacvtapPassthroughNICs {
		dataNIC := interfaces[i+1]
		if got := dataNIC.SelectAttrValue("type", ""); got != "direct" {
			t.Errorf("第 %d 块数据网卡 type = %q，期望 direct", i+1, got)
		}
		source := dataNIC.FindElement("source")
		if source == nil {
			t.Fatalf("第 %d 块数据网卡缺少 <source>", i+1)
		}
		if got := source.SelectAttrValue("dev", ""); got != wantDev {
			t.Errorf("第 %d 块数据网卡 dev = %q，期望 %q", i+1, got, wantDev)
		}
		if got := source.SelectAttrValue("mode", ""); got != "passthrough" {
			t.Errorf("第 %d 块数据网卡 mode = %q，期望 passthrough", i+1, got)
		}
	}
}

func TestBuildCreatePlanISO(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "base.xml")
	if err := os.WriteFile(xmlPath, []byte(testBaseDomainXML), 0644); err != nil {
		t.Fatal(err)
	}

	spec := CreateSpec{
		Name:       "iso-vm",
		VCPU:       1,
		MaxVCPU:    1,
		Memory:     1 << 30,
		MaxMemory:  1 << 30,
		RootVolume: 10 << 30,
		ISO:        "/var/lib/libvirt/images/install.iso",
		Pool:       "default",
		XMLPath:    xmlPath,
	}

	plan, err := buildCreatePlan(spec)
	if err != nil {
		t.Fatalf("buildCreatePlan 返回错误: %v", err)
	}
	if plan.Disks[0].Backing != "" {
		t.Errorf("ISO 安装时根盘不应有 backing，实际 %q", plan.Disks[0].Backing)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromString(plan.XML); err != nil {
		t.Fatalf("生成的 XML 无法解析: %v", err)
	}
	root := doc.Root()

	var cdrom *etree.Element
	for _, disk := range root.FindElements("devices/disk") {
		if disk.SelectAttrValue("device", "") == "cdrom" {
			cdrom = disk
			break
		}
	}
	if cdrom == nil {
		t.Fatal("生成的 XML 缺少 cdrom")
	}
	if got := cdrom.FindElement("source").SelectAttrValue("file", ""); got != spec.ISO {
		t.Errorf("cdrom source = %q，期望 %q", got, spec.ISO)
	}

	boots := root.FindElements("os/boot")
	if len(boots) != 2 {
		t.Fatalf("boot 数量 = %d，期望 2", len(boots))
	}
	if got := boots[0].SelectAttrValue("dev", ""); got != "cdrom" {
		t.Errorf("第一启动项 = %q，期望 cdrom", got)
	}
	if got := boots[1].SelectAttrValue("dev", ""); got != "hd" {
		t.Errorf("第二启动项 = %q，期望 hd", got)
	}
}

func TestVolumeXML(t *testing.T) {
	doc := volumeXML(diskPlan{
		Path:     "/var/lib/libvirt/images/test.qcow2",
		Capacity: 20 << 30,
		Backing:  "/var/lib/libvirt/images/base.qcow2",
	})

	for _, want := range []string{
		"<name>test.qcow2</name>",
		"<capacity>21474836480</capacity>",
		"<allocation>0</allocation>",
		"<backingStore>",
		"<path>/var/lib/libvirt/images/base.qcow2</path>",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("volume XML 缺少 %q:\n%s", want, doc)
		}
	}
}

func TestDeriveDiskPath(t *testing.T) {
	tests := []struct {
		oldPath string
		index   int
		want    string
	}{
		{"/var/lib/libvirt/images/base-vm.qcow2", 0, "/var/lib/libvirt/images/test-vm.qcow2"},
		{"/var/lib/libvirt/images/base-vm-home.qcow2", 1, "/var/lib/libvirt/images/test-vm-home.qcow2"},
		{"/var/lib/libvirt/images/other.qcow2", 0, "/var/lib/libvirt/images/test-vm.qcow2"},
		{"/var/lib/libvirt/images/other.qcow2", 2, "/var/lib/libvirt/images/test-vm-2.qcow2"},
	}

	for _, test := range tests {
		if got := deriveDiskPath(test.oldPath, "base-vm", "test-vm", test.index); got != test.want {
			t.Errorf("deriveDiskPath(%q, %d) = %q，期望 %q", test.oldPath, test.index, got, test.want)
		}
	}
}

func TestBuildCreatePlanSeed(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "base.xml")
	if err := os.WriteFile(xmlPath, []byte(testBaseDomainXML), 0644); err != nil {
		t.Fatal(err)
	}

	spec := CreateSpec{
		Name:          "seed-vm",
		VCPU:          1,
		MaxVCPU:       1,
		Memory:        1 << 30,
		MaxMemory:     1 << 30,
		RootVolume:    10 << 30,
		IP:            "10.10.22.180/23",
		Gateway:       "10.10.22.1",
		DNS:           []string{"10.10.22.1"},
		MgmtNICBridge: "br0",
		BaseImage:     "/base.qcow2",
		Pool:          "default",
		XMLPath:       xmlPath,
	}

	plan, err := buildCreatePlan(spec)
	if err != nil {
		t.Fatalf("buildCreatePlan 返回错误: %v", err)
	}
	if plan.CloudInit == nil {
		t.Fatal("配置 ip 后应生成 cloud-init seed")
	}
	if plan.CloudInit.MAC != plan.MAC {
		t.Errorf("seed MAC = %q，期望 %q", plan.CloudInit.MAC, plan.MAC)
	}
	if plan.CloudInit.NICName != "enp1s0" {
		t.Errorf("seed NICName = %q，期望默认 enp1s0", plan.CloudInit.NICName)
	}
	if !strings.Contains(plan.CloudInit.MetaData(), "instance-id: \"seed-vm\"") {
		t.Errorf("meta-data 缺少 instance-id: %s", plan.CloudInit.MetaData())
	}
	network := plan.CloudInit.NetworkConfig()
	for _, want := range []string{plan.MAC, "10.10.22.180/23", "10.10.22.1", "dhcp4: false", "enp1s0:", "set-name: enp1s0", "to: 0.0.0.0/0"} {
		if !strings.Contains(network, want) {
			t.Errorf("network-config 缺少 %q:\n%s", want, network)
		}
	}
	if strings.Contains(plan.CloudInit.UserData(), "manage_etc_hosts") {
		t.Errorf("user-data 不应设置 manage_etc_hosts: %s", plan.CloudInit.UserData())
	}
}

func TestAttachCloudInitSeed(t *testing.T) {
	xml := `<domain>
  <name>demo</name>
  <devices>
    <controller type='scsi' index='0' model='virtio-scsi'/>
    <disk type='file' device='disk'>
      <source file='/var/lib/libvirt/images/demo.qcow2'/>
      <target dev='vda' bus='virtio'/>
    </disk>
  </devices>
</domain>`

	got, err := attachCloudInitSeed(xml, "/var/lib/libvirt/images/demo-seed.iso")
	if err != nil {
		t.Fatalf("attachCloudInitSeed 返回错误: %v", err)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromString(got); err != nil {
		t.Fatalf("生成的 XML 无法解析: %v", err)
	}
	var cdrom *etree.Element
	for _, disk := range doc.Root().FindElements("devices/disk") {
		if disk.SelectAttrValue("device", "") == "cdrom" {
			cdrom = disk
			break
		}
	}
	if cdrom == nil {
		t.Fatal("生成的 XML 缺少 seed cdrom")
	}
	if got := cdrom.FindElement("source").SelectAttrValue("file", ""); got != "/var/lib/libvirt/images/demo-seed.iso" {
		t.Errorf("seed source = %q", got)
	}
	if got := cdrom.FindElement("target").SelectAttrValue("bus", ""); got != "scsi" {
		t.Errorf("seed bus = %q，期望 scsi", got)
	}
	if cdrom.FindElement("readonly") == nil {
		t.Error("seed cdrom 缺少 readonly")
	}
	if len(doc.Root().FindElements("devices/disk")) != 2 {
		t.Errorf("disk 数量 = %d，期望 2", len(doc.Root().FindElements("devices/disk")))
	}
}
