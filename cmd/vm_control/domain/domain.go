package domain

import (
	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/domain/snapshot"
	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/flags"
	"github.com/spf13/cobra"
)

func CreateCommand() *cobra.Command {
	domainCmd := &cobra.Command{
		Use:   "domain",
		Short: "虚拟机管理",
		// PersistentPreRun: productsPersistentPreRun,
	}

	domainCmd.PersistentFlags().StringSliceVar(&flags.DF.DomainsName, "domains", nil, "虚拟机列表")

	domainCmd.AddCommand(
		listCommand(),
		shutdownCommand(),
		snapshot.CreateCommand(),
	)

	return domainCmd
}
