package snapshot

import (
	"fmt"
	"os"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/flags"
	"github.com/DesistDaydream/go-libvirt/pkg/handler"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"libvirt.org/go/libvirt"
)

func listCommand() *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "列出所有快照",
		Run:   runList,
	}

	return listCmd
}

func runList(cmd *cobra.Command, args []string) {
	if flags.DF.DomainsName != nil {
		domains := handler.FindNeedHandleDomains(flags.DF.DomainsName)
		for _, d := range domains {
			logrus.Infof("%v 在 %v 服务器上，有如下几个快照:", d.DomainName, d.HostName)
			snapshots, _ := d.VirDomain.ListAllSnapshots(0)
			for _, s := range snapshots {
				sName, _ := s.GetName()
				logrus.Infof(sName)
			}
		}
	} else {
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

}
