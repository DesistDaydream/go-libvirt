package snapshot

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/app"
	"github.com/DesistDaydream/go-libvirt/pkg/virt"
)

func deleteCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete",
		Aliases: []string{"del"},
		Short:   "删除快照",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runDelete(a)
		},
	}

	cmd.Flags().BoolVar(&a.Flags.SnapshotChildren, "children", false, "连同子快照一起删除")

	return cmd
}

func runDelete(a *app.App) error {
	if a.Flags.SnapshotName == "" {
		return errors.New("请用 --snapshot 指定快照名称")
	}

	selection, err := a.Select(true)
	if err != nil {
		return err
	}

	return a.RunBatch("snapshot delete "+a.Flags.SnapshotName, selection, func(d *virt.Domain) (string, error) {
		return "", virt.DeleteSnapshot(d, a.Flags.SnapshotName, a.Flags.SnapshotChildren)
	})
}
