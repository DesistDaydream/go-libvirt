package snapshot

import (
	"fmt"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/app"
	"github.com/DesistDaydream/go-libvirt/pkg/virt"
)

func listCommand(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出快照",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runList(a)
		},
	}
}

func runList(a *app.App) error {
	opts := a.SelectOptions()
	if !opts.All && len(opts.Names) == 0 {
		opts.All = true
	}

	selection, err := virt.Select(a.Pool.Hosts(), opts)
	if err != nil {
		return err
	}
	defer virt.FreeDomains(selection.Domains)

	table := tablewriter.NewTable(a.Out, tablewriter.WithHeaderAutoFormat(tw.Off))
	table.Header("宿主机", "虚拟机", "快照", "创建时间", "当前", "描述")

	count := 0
	for _, d := range selection.Domains {
		snapshots, err := virt.ListSnapshots(d)
		if err != nil {
			return fmt.Errorf("列出 %s/%s 的快照失败: %w", d.Host.Name, d.Name, err)
		}
		for _, s := range snapshots {
			created := "-"
			if !s.CreationTime.IsZero() {
				created = s.CreationTime.Format("2006-01-02 15:04:05")
			}
			row := []string{
				d.Host.Name,
				d.Name,
				s.Name,
				created,
				yesNo(s.Current),
				s.Description,
			}
			if err := table.Append(row); err != nil {
				return err
			}
			count++
		}
	}

	if err := table.Render(); err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "共 %d 个快照\n", count)
	for _, pattern := range selection.Unmatched {
		logrus.Warnf("未找到匹配的虚拟机: %s", pattern)
	}
	return nil
}

func yesNo(value bool) string {
	if value {
		return "是"
	}
	return "否"
}
