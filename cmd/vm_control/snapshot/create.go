package snapshot

import (
	"fmt"

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
	if snapshotFlags.SnapshotName == "" {
		logrus.Fatal("请指定快照名称")
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

	for _, conn := range handler.Conns {
		// 在每个 libvirtd 的连接中寻找要创建快照的 Domain
		// TODO: 有没有更好的写法在多个目标中寻找多个目标？
		for _, dName := range snapshotFlags.DomainName {
			domain, err := conn.LookupDomainByName(dName)
			if err != nil {
				logrus.Errorf("未找到 【%v】 虚拟机", dName)
			}

			dName, _ := domain.GetName()

			logrus.Debugf("开始处理 %v", dName)

			domainSnapshot, err := domain.CreateSnapshotXML(xml, 0)
			if err != nil {
				logrus.Errorf("创建快照失败，原因: %v", err)
				return
			}

			sName, _ := domainSnapshot.GetName()

			logrus.Debugf("检查创建的快照名称: %v", sName)
		}
	}
}
