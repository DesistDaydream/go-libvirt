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

// 不先获取需要处理的虚拟机，而是找到一个处理一个
// TODO: 这种方式和先找到再处理哪个合适呢
// func runDelete(cmd *cobra.Command, args []string) {
// 	if snapshotFlags.SnapshotName == "" || snapshotFlags.DomainName == nil {
// 		logrus.Fatal("请指定快照名称和虚拟机名称")
// 	}
// 	for _, conn := range handler.Conns {
// 		hostname, _ := conn.GetHostname()
// 		for dNameIndex, dName := range snapshotFlags.DomainName {
// 			domain, _ := conn.LookupDomainByName(dName)
// 			if domain != nil {
// 				logrus.Infof("在 %v 上找到虚拟机 %v", hostname, dName)
// 				snapshot, err := domain.SnapshotLookupByName(snapshotFlags.SnapshotName, 0)
// 				if err != nil {
// 					logrus.Errorf("无法在【%v】中找到【%v】快照，原因: %v", dName, snapshotFlags.SnapshotName, err)
// 					continue
// 				}
// 				defer snapshot.Free()
// 				err = snapshot.Delete(libvirt.DOMAIN_SNAPSHOT_DELETE_CHILDREN)
// 				if err != nil {
// 					logrus.Errorf("删除快照失败，原因: %v", err)
// 					continue
// 				} else {
// 					logrus.Infof("已成功删除 %v 中的 %v 快照", dName, snapshotFlags.SnapshotName)
// 					// TODO: 删除 snapshotFlags.DomainName 中的 dName 元素
// 					snapshotFlags.DomainName = append(snapshotFlags.DomainName[:dNameIndex], snapshotFlags.DomainName[dNameIndex+1:]...)
// 				}
// 				domain.Free()
// 			}
// 		}
// 	}
// }
