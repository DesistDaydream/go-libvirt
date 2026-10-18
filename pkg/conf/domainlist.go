package conf

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/DesistDaydream/go-libvirt/pkg/virt"
)

// ByteSize 解析 4GiB、20G、1024MiB 这类容量写法。
type ByteSize uint64

func (b *ByteSize) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode {
		return errors.New("容量必须是标量")
	}
	value, err := parseByteSize(node.Value)
	if err != nil {
		return err
	}
	*b = ByteSize(value)
	return nil
}

var sizePattern = regexp.MustCompile(`(?i)^\s*([0-9]+(?:\.[0-9]+)?)\s*(B|K|KB|KIB|M|MB|MIB|G|GB|GIB|T|TB|TIB)?\s*$`)

func parseByteSize(s string) (uint64, error) {
	match := sizePattern.FindStringSubmatch(s)
	if match == nil {
		return 0, fmt.Errorf("无法解析容量 %q（示例：4GiB、20G、1024MiB）", s)
	}

	value, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0, fmt.Errorf("无法解析容量 %q: %w", s, err)
	}

	var multiplier float64
	switch strings.ToUpper(match[2]) {
	case "", "B":
		multiplier = 1
	case "K", "KB", "KIB":
		multiplier = 1 << 10
	case "M", "MB", "MIB":
		multiplier = 1 << 20
	case "G", "GB", "GIB":
		multiplier = 1 << 30
	case "T", "TB", "TIB":
		multiplier = 1 << 40
	default:
		return 0, fmt.Errorf("无法解析容量单位 %q", match[2])
	}

	bytes := value * multiplier
	if bytes < 1 {
		return 0, fmt.Errorf("容量必须大于 0: %q", s)
	}
	return uint64(bytes), nil
}

type DomainList struct {
	Domains []DomainSpec `yaml:"domains"`
}

// VolumeSpec 是额外数据盘，可以写成 "/data" 或 {path: /data, size: 50GiB}。
type VolumeSpec struct {
	Path string
	Size ByteSize
}

func (v *VolumeSpec) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		v.Path = node.Value
		return nil
	}
	if node.Kind != yaml.MappingNode {
		return errors.New("volume 必须是路径或 {path, size}")
	}
	type plain VolumeSpec
	return node.Decode((*plain)(v))
}

// DomainSpec 是 domain-list 文件中的一台虚拟机。
type DomainSpec struct {
	Name                   string       `yaml:"name"`
	Host                   string       `yaml:"host"`
	VCPU                   int          `yaml:"vcpu"`
	MaxVCPU                int          `yaml:"max_vcpu"`
	Memory                 ByteSize     `yaml:"memory"`
	MaxMemory              ByteSize     `yaml:"max_memory"`
	RootVolume             ByteSize     `yaml:"root_volume"`
	Volumes                []VolumeSpec `yaml:"volumes"`
	IP                     string       `yaml:"ip"`
	MgmtNICBridge          string       `yaml:"mgmt_nic_bridge"`
	MacvtapPassthroughNICs []string     `yaml:"macvtap_passthrough_nics"`
	PCIPassthroughNICs     []string     `yaml:"pci_passthrough_nics"`
	XML                    string       `yaml:"xml"`
	DomainImage            string       `yaml:"domain_image"`
	BaseImage              string       `yaml:"base_image"`
	ISO                    string       `yaml:"iso"`
	Pool                   string       `yaml:"pool"`
}

func LoadDomainList(path string) (*DomainList, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 domain-list 文件失败: %w", err)
	}

	var list DomainList
	if err := yaml.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("解析 domain-list 文件失败: %w", err)
	}
	if len(list.Domains) == 0 {
		return nil, fmt.Errorf("domain-list 文件 %s 中没有 domains", path)
	}

	dir := filepath.Dir(path)
	seen := make(map[string]bool, len(list.Domains))
	for i := range list.Domains {
		spec := &list.Domains[i]
		if spec.XML != "" && !filepath.IsAbs(spec.XML) {
			spec.XML = filepath.Join(dir, spec.XML)
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
		if err := spec.validate(); err != nil {
			return nil, fmt.Errorf("domains[%d]（%s）: %w", i, spec.Name, err)
		}
		if seen[spec.Name] {
			return nil, fmt.Errorf("虚拟机名称 %q 重复", spec.Name)
		}
		seen[spec.Name] = true
	}

	return &list, nil
}

func (s *DomainSpec) validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return errors.New("name 不能为空")
	}
	if s.VCPU <= 0 {
		return errors.New("vcpu 必须大于 0")
	}
	if s.MaxVCPU < s.VCPU {
		return errors.New("max_vcpu 不能小于 vcpu")
	}
	if s.Memory == 0 {
		return errors.New("memory 必须大于 0")
	}
	if s.MaxMemory < s.Memory {
		return errors.New("max_memory 不能小于 memory")
	}
	if s.RootVolume == 0 {
		return errors.New("root_volume 必须大于 0")
	}
	for _, volume := range s.Volumes {
		if strings.TrimSpace(volume.Path) == "" {
			return errors.New("volumes 中的挂载路径不能为空")
		}
	}
	for _, pci := range s.PCIPassthroughNICs {
		if strings.TrimSpace(pci) == "" {
			return errors.New("pci_passthrough_nics 中的 PCI 地址不能为空")
		}
	}
	seenMacvtapPassthroughNICs := make(map[string]bool, len(s.MacvtapPassthroughNICs))
	for _, nic := range s.MacvtapPassthroughNICs {
		if strings.TrimSpace(nic) == "" {
			return errors.New("macvtap_passthrough_nics 中的网卡名不能为空")
		}
		if seenMacvtapPassthroughNICs[nic] {
			return fmt.Errorf("macvtap_passthrough_nics 中重复指定网卡 %s", nic)
		}
		seenMacvtapPassthroughNICs[nic] = true
	}
	if len(s.MacvtapPassthroughNICs) > 0 && len(s.PCIPassthroughNICs) > 0 {
		return errors.New("macvtap_passthrough_nics 与 pci_passthrough_nics 不能同时使用")
	}
	if s.XML == "" {
		return errors.New("xml 不能为空")
	}
	if _, err := os.Stat(s.XML); err != nil {
		return fmt.Errorf("基础 XML 不可用: %w", err)
	}
	if s.IP != "" {
		ip := net.ParseIP(s.IP)
		if ip == nil || ip.To4() == nil {
			return fmt.Errorf("ip %q 不是 IPv4 地址", s.IP)
		}
	}

	sources := 0
	for _, source := range []string{s.DomainImage, s.BaseImage, s.ISO} {
		if source != "" {
			sources++
		}
	}
	if sources != 1 {
		return errors.New("domain_image、base_image、iso 必须且只能指定一个")
	}
	if s.DomainImage != "" && !filepath.IsAbs(s.DomainImage) {
		return errors.New("domain_image 必须是远端绝对路径")
	}
	if s.BaseImage != "" && !filepath.IsAbs(s.BaseImage) {
		return errors.New("base_image 必须是远端绝对路径")
	}
	if s.ISO != "" && !filepath.IsAbs(s.ISO) {
		return errors.New("iso 必须是远端绝对路径")
	}
	return nil
}

func (l *DomainList) Find(name string) (*DomainSpec, error) {
	for i := range l.Domains {
		if l.Domains[i].Name == name {
			return &l.Domains[i], nil
		}
	}

	names := make([]string, 0, len(l.Domains))
	for _, spec := range l.Domains {
		names = append(names, spec.Name)
	}
	return nil, fmt.Errorf("domain-list 中没有 %q（可用: %s）", name, strings.Join(names, ", "))
}

func (s *DomainSpec) ToCreateSpec() virt.CreateSpec {
	return virt.CreateSpec{
		Name:                   s.Name,
		VCPU:                   s.VCPU,
		MaxVCPU:                s.MaxVCPU,
		Memory:                 uint64(s.Memory),
		MaxMemory:              uint64(s.MaxMemory),
		RootVolume:             uint64(s.RootVolume),
		Volumes:                toCreateVolumes(s.Volumes, s.RootVolume),
		IP:                     s.IP,
		MgmtNICBridge:          s.MgmtNICBridge,
		MacvtapPassthroughNICs: s.MacvtapPassthroughNICs,
		PCIPassthroughNICs:     s.PCIPassthroughNICs,
		BaseImage:              s.BaseImage,
		DomainImage:            s.DomainImage,
		ISO:                    s.ISO,
		Pool:                   s.Pool,
		XMLPath:                s.XML,
	}
}

func toCreateVolumes(volumes []VolumeSpec, defaultSize ByteSize) []virt.CreateVolume {
	out := make([]virt.CreateVolume, 0, len(volumes))
	for _, volume := range volumes {
		size := volume.Size
		if size == 0 {
			size = defaultSize
		}
		out = append(out, virt.CreateVolume{Path: volume.Path, Size: uint64(size)})
	}
	return out
}
