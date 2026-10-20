package domain

import (
	"github.com/spf13/cobra"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/app"
	"github.com/DesistDaydream/go-libvirt/pkg/conf"
	"github.com/DesistDaydream/go-libvirt/pkg/virt"
)

var longHelp string = `根据 domain-list.yaml 创建一台虚拟机。

根分区来源三选一（domain_image、base_image、iso 互斥）：
  domain_image  先把该镜像完整拷贝到 <存储目录>/<名称>.qcow2，再用它作为根盘。最后以导入镜像方式安装虚拟机
  base_image    创建新的 qcow2 overlay，以 base_image 为 backing file。最后以导入镜像方式安装虚拟机
  iso           创建空 qcow2，挂载 ISO 并从光盘启动安装

硬盘：
  root_volume   根盘大小；volumes 未指定 size 时默认继承它
  volumes       额外空盘，文件名 <name>-<挂载路径>.qcow2，按 vdb/vdc… 顺序挂载

网络与直通：
  mgmt_nic_bridge  管理网卡使用的网桥
  ip               静态 IPv4 CIDR（如 10.10.22.180/23）
  mgmt_nic_name    guest 内管理网卡名，默认 enp1s0
  gateway          静态 IP 的网关
  dns              静态 IP 的 DNS，可重复
  macvtap_passthrough_nics 宿主机镜像口列表，每个口增加一块 virtio（macvtap passthrough）；与 pci_passthrough_nics 互斥
  pci_passthrough_nics     宿主机 PCI 地址列表，整卡直通；创建前会检查 IOMMU group

配置 ip 后会生成 cloud-init NoCloud seed ISO 并挂成 cdrom；基础镜像里必须已安装并启用 cloud-init，
且不要再写死静态网卡配置。

示例：
  vm-control --ips 10.10.22.62 domain create -f domain-list.yaml hdf-01
  vm-control --ips 10.10.22.62 domain create -f domain-list.yaml --dry-run hdf-01
  vm-control --ips 10.10.22.62 host nic list

--dry-run 只打印生成的 XML；--no-start 只 define 不启动。
`

func createCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "从 domain list 文件创建虚拟机",
		Long:  longHelp,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runCreate(a, args[0])
		},
	}

	pf := cmd.Flags()
	pf.StringVarP(&a.Flags.DomainFile, "file", "f", "domain-list.yaml", "domain list 文件")
	pf.BoolVar(&a.Flags.NoStart, "no-start", false, "只定义虚拟机，不启动")
	pf.BoolVar(&a.Flags.DryRun, "dry-run", false, "只打印生成的 XML，不创建")

	return cmd
}

func runCreate(a *app.App, name string) error {
	list, err := conf.LoadDomainList(a.Flags.DomainFile)
	if err != nil {
		return err
	}

	spec, err := list.Find(name)
	if err != nil {
		return err
	}

	host, err := a.HostFor(spec.Host)
	if err != nil {
		return err
	}

	return virt.CreateDomain(host, spec.ToCreateSpec(), virt.CreateOptions{
		Start:  !a.Flags.NoStart,
		DryRun: a.Flags.DryRun,
		Out:    a.Out,
	})
}
