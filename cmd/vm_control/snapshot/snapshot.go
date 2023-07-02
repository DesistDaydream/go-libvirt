package snapshot

import (
	"github.com/spf13/cobra"
)

type SnapshotFlags struct {
	DomainName   []string
	SnapshotName string
}

var snapshotFlags SnapshotFlags

func CreateCommand() *cobra.Command {
	snapshotCmd := &cobra.Command{
		Use:   "snapshot",
		Short: "快照管理",
		// PersistentPreRun: productsPersistentPreRun,
	}

	snapshotCmd.PersistentFlags().StringSliceVar(&snapshotFlags.DomainName, "domain", nil, "虚拟机列表")
	snapshotCmd.PersistentFlags().StringVar(&snapshotFlags.SnapshotName, "snapshot", "", "快照名称")

	snapshotCmd.AddCommand(
		listCommand(),
		createCommand(),
	)

	return snapshotCmd
}
