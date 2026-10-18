package virt

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"libvirt.org/go/libvirt"
)

// Domain 是一台虚拟机句柄，绑定到 Pool 中某条仍然存活的连接。
type Domain struct {
	Host *Host
	Name string
	Vir  *libvirt.Domain
}

func (d *Domain) Free() {
	if d.Vir != nil {
		_ = d.Vir.Free()
		d.Vir = nil
	}
}

func FreeDomains(domains []*Domain) {
	for _, d := range domains {
		d.Free()
	}
}

// SelectOptions 描述一次批量操作要选哪些虚拟机。
type SelectOptions struct {
	// Names 是名称或 glob 模式；literal 名称精确匹配。
	Names []string
	// All 选择所有宿主机上的所有虚拟机。
	All bool
	// Running 只保留运行中的虚拟机（含 paused/blocked）。
	Running bool
	// Stopped 只保留非运行状态的虚拟机。
	Stopped bool
}

// Selection 是选择结果。Unmatched 中的模式一台虚拟机也没匹配到。
type Selection struct {
	Domains   []*Domain
	Unmatched []string
}

// AllDomains 列出所有宿主机上的全部虚拟机。
func (p *Pool) AllDomains() ([]*Domain, error) {
	return listDomains(p.hosts)
}

func listDomains(hosts []*Host) ([]*Domain, error) {
	var domains []*Domain
	for _, h := range hosts {
		virDomains, err := h.Conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
		if err != nil {
			FreeDomains(domains)
			return nil, fmt.Errorf("列出 %s 上的虚拟机失败: %w", h.Name, err)
		}

		for i := range virDomains {
			name, err := virDomains[i].GetName()
			if err != nil {
				_ = virDomains[i].Free()
				for j := i + 1; j < len(virDomains); j++ {
					_ = virDomains[j].Free()
				}
				FreeDomains(domains)
				return nil, fmt.Errorf("读取 %s 上的虚拟机名称失败: %w", h.Name, err)
			}
			domains = append(domains, &Domain{Host: h, Name: name, Vir: &virDomains[i]})
		}
	}
	return domains, nil
}

// Select 按名称、glob 与状态筛选虚拟机。未被选中的句柄会在函数内释放。
func Select(hosts []*Host, opts SelectOptions) (*Selection, error) {
	all, err := listDomains(hosts)
	if err != nil {
		return nil, err
	}
	return chooseDomains(all, opts)
}

func chooseDomains(all []*Domain, opts SelectOptions) (*Selection, error) {
	if opts.All && len(opts.Names) > 0 {
		FreeDomains(all)
		return nil, errors.New("--all 与 --domains 不能同时使用")
	}
	if opts.Running && opts.Stopped {
		FreeDomains(all)
		return nil, errors.New("--running 与 --stopped 不能同时使用")
	}

	sel := &Selection{}
	if opts.All {
		sel.Domains = all
	} else {
		selected := make(map[*Domain]bool)
		for _, pattern := range dedupeStrings(opts.Names) {
			matched := false
			for _, d := range all {
				ok, err := matchName(pattern, d.Name)
				if err != nil {
					FreeDomains(all)
					return nil, err
				}
				if !ok || selected[d] {
					continue
				}
				matched = true
				selected[d] = true
				sel.Domains = append(sel.Domains, d)
			}
			if !matched {
				sel.Unmatched = append(sel.Unmatched, pattern)
			}
		}
		for _, d := range all {
			if !selected[d] {
				d.Free()
			}
		}
	}

	if opts.Running || opts.Stopped {
		keep := make([]bool, len(sel.Domains))
		for i, d := range sel.Domains {
			active, err := d.Vir.IsActive()
			if err != nil {
				FreeDomains(sel.Domains)
				return nil, fmt.Errorf("读取 %s/%s 状态失败: %w", d.Host.Name, d.Name, err)
			}
			keep[i] = (opts.Running && active) || (opts.Stopped && !active)
		}

		kept := make([]*Domain, 0, len(sel.Domains))
		for i, d := range sel.Domains {
			if keep[i] {
				kept = append(kept, d)
			} else {
				d.Free()
			}
		}
		sel.Domains = kept
	}

	return sel, nil
}

func matchName(pattern, name string) (bool, error) {
	if !strings.ContainsAny(pattern, "*?[") {
		return pattern == name, nil
	}
	ok, err := path.Match(pattern, name)
	if err != nil {
		return false, fmt.Errorf("无效的匹配模式 %q: %w", pattern, err)
	}
	return ok, nil
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

// StateText 返回虚拟机状态的中文名称。
func StateText(state libvirt.DomainState) string {
	switch state {
	case libvirt.DOMAIN_RUNNING:
		return "运行"
	case libvirt.DOMAIN_BLOCKED:
		return "阻塞"
	case libvirt.DOMAIN_PAUSED:
		return "暂停"
	case libvirt.DOMAIN_SHUTDOWN:
		return "关机中"
	case libvirt.DOMAIN_SHUTOFF:
		return "关机"
	case libvirt.DOMAIN_CRASHED:
		return "崩溃"
	case libvirt.DOMAIN_PMSUSPENDED:
		return "休眠"
	default:
		return "未知"
	}
}
