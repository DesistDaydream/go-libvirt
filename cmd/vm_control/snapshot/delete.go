package snapshot

import (
	"github.com/DesistDaydream/go-libvirt/pkg/handler"
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
	for _, conn := range handler.Conns {
		for dNameIndex, dName := range snapshotFlags.DomainName {
			domain, err := conn.LookupDomainByName(dName)
			if err != nil {
				logrus.Errorf("未找到【%v】虚拟机", dName)
				continue
			} else {
				snapshot, err := domain.SnapshotLookupByName(snapshotFlags.SnapshotName, 0)
				if err != nil {
					logrus.Errorf("无法在【%v】中找到【%v】快照，原因: %v", dName, snapshotFlags.SnapshotName, err)
					continue
				}
				defer snapshot.Free()

				err = snapshot.Delete(libvirt.DOMAIN_SNAPSHOT_DELETE_CHILDREN)
				if err != nil {
					logrus.Errorf("删除快照失败，原因: %v", err)
					continue
				} else {
					logrus.Infof("已成功删除 %v 中的 %v 快照", dName, snapshotFlags.SnapshotName)
					// TODO: 删除 snapshotFlags.DomainName 中的 dName 元素
					snapshotFlags.DomainName = append(snapshotFlags.DomainName[:dNameIndex], snapshotFlags.DomainName[dNameIndex+1:]...)
				}
			}
			defer domain.Free()
		}
	}
}
