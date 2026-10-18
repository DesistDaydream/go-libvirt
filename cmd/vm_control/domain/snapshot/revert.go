package snapshot

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/app"
	"github.com/DesistDaydream/go-libvirt/pkg/virt"
)

func revertCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revert",
		Short: "回滚到快照",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runRevert(a)
		},
	}

	pf := cmd.Flags()
	pf.BoolVar(&a.Flags.SnapshotForce, "force", false, "强制回滚")
	pf.BoolVar(&a.Flags.SnapshotRunning, "running", false, "回滚后保持运行")
	pf.BoolVar(&a.Flags.SnapshotPaused, "paused", false, "回滚后暂停")

	return cmd
}

func runRevert(a *app.App) error {
	if a.Flags.SnapshotName == "" {
		return errors.New("请用 --snapshot 指定快照名称")
	}
	if a.Flags.SnapshotRunning && a.Flags.SnapshotPaused {
		return errors.New("--running 与 --paused 不能同时使用")
	}

	selection, err := a.Select(true)
	if err != nil {
		return err
	}

	return a.RunBatch("snapshot revert "+a.Flags.SnapshotName, selection, func(d *virt.Domain) (string, error) {
		return "", virt.RevertSnapshot(d, a.Flags.SnapshotName, a.Flags.SnapshotForce, a.Flags.SnapshotRunning, a.Flags.SnapshotPaused)
	})
}
