package snapshot

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/app"
	"github.com/DesistDaydream/go-libvirt/pkg/virt"
)

func createCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "创建内部快照",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runCreate(a)
		},
	}

	pf := cmd.Flags()
	pf.StringVar(&a.Flags.SnapshotDescription, "description", "", "快照描述")
	pf.BoolVar(&a.Flags.SnapshotMemory, "memory", false, "同时保存内存状态（内部快照）")
	pf.BoolVar(&a.Flags.SnapshotQuiesce, "quiesce", false, "要求 guest agent 冻结文件系统")

	return cmd
}

func runCreate(a *app.App) error {
	if a.Flags.SnapshotName == "" {
		return errors.New("请用 --snapshot 指定快照名称")
	}

	selection, err := a.Select(true)
	if err != nil {
		return err
	}

	opts := virt.SnapshotOptions{
		Name:        a.Flags.SnapshotName,
		Description: a.Flags.SnapshotDescription,
		Memory:      a.Flags.SnapshotMemory,
		Quiesce:     a.Flags.SnapshotQuiesce,
	}

	return a.RunBatch("snapshot create "+a.Flags.SnapshotName, selection, func(d *virt.Domain) (string, error) {
		if _, err := virt.CreateSnapshot(d, opts); err != nil {
			return "", err
		}
		return "", nil
	})
}
