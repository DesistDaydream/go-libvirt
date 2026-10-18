package domain

import (
	"fmt"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/app"
	"github.com/DesistDaydream/go-libvirt/pkg/virt"
)

func listCommand(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出所有虚拟机",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runList(a)
		},
	}
}

func runList(a *app.App) error {
	domains, err := a.Pool.AllDomains()
	if err != nil {
		return err
	}
	defer virt.FreeDomains(domains)

	for _, host := range a.Pool.Hosts() {
		fmt.Fprintf(a.Out, "======== %s ========\n", host.Name)

		table := tablewriter.NewTable(a.Out, tablewriter.WithHeaderAutoFormat(tw.Off))
		table.Header("实例", "状态", "vCPU", "内存(GiB)", "自动启动")

		var allocatedCPU uint
		var allocatedMem uint64
		for _, d := range domains {
			if d.Host != host {
				continue
			}

			info, err := d.Vir.GetInfo()
			if err != nil {
				return fmt.Errorf("读取 %s/%s 信息失败: %w", host.Name, d.Name, err)
			}
			autostart, _ := d.Vir.GetAutostart()

			row := []string{
				d.Name,
				virt.StateText(info.State),
				strconv.Itoa(int(info.NrVirtCpu)),
				fmt.Sprintf("%.2f", float64(info.MaxMem)/1024/1024),
				yesNo(autostart),
			}
			if err := table.Append(row); err != nil {
				return err
			}

			allocatedCPU += info.NrVirtCpu
			allocatedMem += info.MaxMem
		}

		if err := table.Render(); err != nil {
			return err
		}

		node, err := host.Conn.GetNodeInfo()
		if err != nil {
			return fmt.Errorf("读取 %s 节点信息失败: %w", host.Name, err)
		}
		fmt.Fprintf(a.Out, "共 %d vCPU，已分配 %d；共 %.2f GiB 内存，已分配 %.2f GiB\n\n",
			node.Cpus, allocatedCPU,
			float64(node.Memory)/1024/1024, float64(allocatedMem)/1024/1024)
	}

	return nil
}

func yesNo(value bool) string {
	if value {
		return "是"
	}
	return "否"
}
