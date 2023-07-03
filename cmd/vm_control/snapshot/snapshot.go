package snapshot

import (
	"github.com/DesistDaydream/go-libvirt/pkg/handler"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"libvirt.org/go/libvirt"
)

type SnapshotFlags struct {
	DomainName   []string
	SnapshotName string
}

var snapshotFlags SnapshotFlags

func CreateCommand() *cobra.Command {
	snapshotCmd := &cobra.Command{
		Use:   "snapshot",
		Short: "快照管理",
		// PersistentPreRun: productsPersistentPreRun,
	}

	snapshotCmd.PersistentFlags().StringSliceVar(&snapshotFlags.DomainName, "domain", nil, "虚拟机列表")
	snapshotCmd.PersistentFlags().StringVar(&snapshotFlags.SnapshotName, "snapshot", "", "快照名称")

	snapshotCmd.AddCommand(
		listCommand(),
		createCommand(),
		deleteCommand(),
	)

	return snapshotCmd
}

type needHandleDomain struct {
	hostName   string
	domainName string
	domain     *libvirt.Domain
}

var nhds []needHandleDomain

func findNeedHandleDomains() []needHandleDomain {
	for _, conn := range handler.Conns {
		hostname, _ := conn.GetHostname()
		for _, dName := range snapshotFlags.DomainName {
			domain, _ := conn.LookupDomainByName(dName)
			if domain != nil {
				nhds = append(nhds, needHandleDomain{
					hostName:   hostname,
					domainName: dName,
					domain:     domain,
				})
			}
		}
	}

	for _, n := range nhds {
		logrus.Debugf("在 %v 上找到需要处理的虚拟机: %v", n.hostName, n.domainName)
	}

	return nhds
}
