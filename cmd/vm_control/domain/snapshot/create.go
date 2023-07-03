package snapshot

import (
	"fmt"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/flags"
	"github.com/DesistDaydream/go-libvirt/pkg/handler"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func createCommand() *cobra.Command {
	var createCmd = &cobra.Command{
		Use:   "create",
		Short: "创建快照",
		Run:   runCreate,
	}

	return createCmd
}

func runCreate(cmd *cobra.Command, args []string) {
	if snapshotFlags.SnapshotName == "" || flags.DF.DomainsName == nil {
		logrus.Fatal("请指定快照名称和虚拟机名称")
	}

	var xml string = fmt.Sprintf(`
<domainsnapshot>
	<name>%s</name>
	<memory snapshot='no'/>
	<disks>
	<disk name='vda' snapshot='internal'/>
	</disks>
</domainsnapshot>
`, snapshotFlags.SnapshotName)

	domains := handler.FindNeedHandleDomains(flags.DF.DomainsName)

	for _, d := range domains {
		dName, _ := d.VirDomain.GetName()
		logrus.Debugf("开始处理 %v", dName)

		domainSnapshot, err := d.VirDomain.CreateSnapshotXML(xml, 0)
		if err != nil {
			logrus.Errorf("创建快照失败，原因: %v", err)
			continue
		} else {
			logrus.Infof("创建快照成功")
			sName, _ := domainSnapshot.GetName()
			logrus.Debugf("检查创建的快照名称: %v", sName)
		}
	}
}
