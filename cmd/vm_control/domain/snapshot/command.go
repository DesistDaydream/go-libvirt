package snapshot

import (
	"github.com/spf13/cobra"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/app"
)

func CreateCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "快照管理",
	}

	cmd.PersistentFlags().StringVarP(&a.Flags.SnapshotName, "snapshot", "s", "", "快照名称（create/delete/revert 必填）")

	cmd.AddCommand(
		listCommand(a),
		createCommand(a),
		deleteCommand(a),
		revertCommand(a),
	)

	return cmd
}
