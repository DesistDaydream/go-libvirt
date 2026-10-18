package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/app"
	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/domain"
	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/host"
)

func main() {
	a := app.New()
	defer a.Close()

	if err := newRootCommand(a).Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCommand(a *app.App) *cobra.Command {
	root := &cobra.Command{
		Use:   "vm-control",
		Short: "KVM/QEMU 虚拟机批量管理",
		Long: `基于 libvirt 的虚拟机批量管理工具。

一次连接多台宿主机，按名称、glob 或状态选择虚拟机，批量执行开关机与
快照操作，并逐台汇总执行结果。连接只建立一次，命令执行期间保持有效。`,
		SilenceUsage: true,
	}

	a.AddRootFlags(root)
	root.AddCommand(domain.CreateCommand(a), host.CreateCommand(a))
	return root
}
