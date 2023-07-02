package snapshot

import (
	"fmt"
	"os"

	"github.com/DesistDaydream/go-libvirt/pkg/handler"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"libvirt.org/go/libvirt"
)

func ListCommand() *cobra.Command {
	var ListCmd = &cobra.Command{
		Use:   "list",
		Short: "列出所有快照",
		Run:   runList,
	}

	return ListCmd
}

func runList(cmd *cobra.Command, args []string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"实例", "快照"})

	for _, conn := range handler.Conns {
		defer conn.Close()

		curHost, _ := conn.GetHostname()
		fmt.Printf("======== %v ========\n", curHost)

		doms, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
		if err != nil {
			logrus.Errorf("无法列出 Domain，原因: %v", err)
		}

		for _, dom := range doms {
			dName, _ := dom.GetName()
			snapshots, _ := dom.ListAllSnapshots(0)
			for _, s := range snapshots {
				sName, _ := s.GetName()
				table.Append([]string{dName, sName})
			}
		}
		table.Render()
	}
}
