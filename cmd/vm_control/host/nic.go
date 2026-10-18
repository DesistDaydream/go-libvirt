package host

import (
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/app"
	"github.com/DesistDaydream/go-libvirt/pkg/virt"
)

func nicCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nic",
		Short: "宿主机网卡管理",
	}
	cmd.AddCommand(nicListCommand(a))
	return cmd
}

func nicListCommand(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出宿主机网卡名称与 PCI 地址",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runNICList(a)
		},
	}
}

func runNICList(a *app.App) error {
	table := tablewriter.NewTable(a.Out, tablewriter.WithHeaderAutoFormat(tw.Off))
	table.Header("宿主机", "网卡", "PCI 地址")

	for _, host := range a.Pool.Hosts() {
		nics, err := virt.HostNICs(host)
		if err != nil {
			return err
		}
		for _, nic := range nics {
			if err := table.Append([]string{host.Name, nic.Name, nic.PCI}); err != nil {
				return err
			}
		}
	}

	return table.Render()
}
