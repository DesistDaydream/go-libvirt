package snapshot

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"libvirt.org/go/libvirt"
)

func deleteCommand() *cobra.Command {
	var deleteCmd = &cobra.Command{
		Use:   "del",
		Short: "删除快照",
		Run:   runDelete,
	}

	return deleteCmd
}

func runDelete(cmd *cobra.Command, args []string) {
	if snapshotFlags.SnapshotName == "" || snapshotFlags.DomainName == nil {
		logrus.Fatal("请指定快照名称和虚拟机名称")
	}

	nhds := findNeedHandleDomains()

	for _, n := range nhds {
		snapshot, err := n.domain.SnapshotLookupByName(snapshotFlags.SnapshotName, 0)
		if err != nil {
			logrus.Errorf("无法在【%v】中找到【%v】快照，原因: %v", n.domainName, snapshotFlags.SnapshotName, err)
			continue
		}
		defer snapshot.Free()

		err = snapshot.Delete(libvirt.DOMAIN_SNAPSHOT_DELETE_CHILDREN)
		if err != nil {
			logrus.Errorf("删除快照失败，原因: %v", err)
			continue
		}

		logrus.Infof("已成功删除 %v 中的 %v 快照", n.domainName, snapshotFlags.SnapshotName)
	}
}
