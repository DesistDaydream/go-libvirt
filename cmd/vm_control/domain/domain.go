package domain

import (
	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/domain/snapshot"
	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var domainCmd *cobra.Command

func CreateCommand() *cobra.Command {
	domainCmd = &cobra.Command{
		Use:   "domain",
		Short: "虚拟机管理",
		// PersistentPreRun: productsPersistentPreRun,
	}

	cobra.OnInitialize(initConfig)

	domainCmd.PersistentFlags().StringSliceVar(&flags.DF.DomainsName, "domains", nil, "虚拟机列表")

	domainCmd.AddCommand(
		listCommand(),
		startCommand(),
		shutdownCommand(),
		snapshot.CreateCommand(),
	)

	return domainCmd
}

func initConfig() {
	viper.BindPFlag("domain.domains", domainCmd.Flags().Lookup("domains"))
	flags.DF.DomainsName = viper.GetStringSlice("domain.domains")
}
