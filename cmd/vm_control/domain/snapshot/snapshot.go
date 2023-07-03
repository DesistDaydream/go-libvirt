package snapshot

import (
	"github.com/spf13/cobra"
)

type SnapshotFlags struct {
	SnapshotName string
}

var snapshotFlags SnapshotFlags

func CreateCommand() *cobra.Command {
	snapshotCmd := &cobra.Command{
		Use:   "snapshot",
		Short: "快照管理",
		// PersistentPreRun: productsPersistentPreRun,
	}

	snapshotCmd.PersistentFlags().StringVar(&snapshotFlags.SnapshotName, "snapshot", "", "快照名称")

	snapshotCmd.AddCommand(
		listCommand(),
		createCommand(),
		deleteCommand(),
	)

	return snapshotCmd
}
