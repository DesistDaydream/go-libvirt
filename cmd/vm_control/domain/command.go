package domain

import (
	"github.com/spf13/cobra"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/app"
	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/domain/snapshot"
)

// CreateCommand 构建 domain 子命令。a.Init 放在这里而不是 root，是为了让
// completion 等不需要 libvirt 的命令不被连接逻辑拖累。
func CreateCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "domain",
		Short:             "虚拟机管理",
		PersistentPreRunE: a.Init,
	}

	pf := cmd.PersistentFlags()
	pf.StringSliceVarP(&a.Flags.Domains, "domains", "d", nil, "虚拟机名称或 glob 模式，逗号分隔，例如 web-*,db-01")
	pf.BoolVar(&a.Flags.All, "all", false, "选择所有虚拟机")
	pf.BoolVar(&a.Flags.Running, "running", false, "只选择运行中的虚拟机")
	pf.BoolVar(&a.Flags.Stopped, "stopped", false, "只选择已关机的虚拟机")
	pf.BoolVar(&a.Flags.Parallel, "parallel", true, "多宿主机并行执行；同一宿主机内串行")

	cmd.AddCommand(
		listCommand(a),
		startCommand(a),
		shutdownCommand(a),
		createCommand(a),
		snapshot.CreateCommand(a),
	)

	return cmd
}
