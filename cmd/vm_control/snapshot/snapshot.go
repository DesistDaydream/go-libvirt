package snapshot

import (
	"github.com/spf13/cobra"
)

// type SnapshotFlags struct {
// }

// var snapshotFlags SnapshotFlags

func CreateCommand() *cobra.Command {
	snapshotCmd := &cobra.Command{
		Use:   "snapshot",
		Short: "快照管理",
		// PersistentPreRun: productsPersistentPreRun,
	}

	snapshotCmd.AddCommand(
		ListCommand(),
	)

	return snapshotCmd
}
