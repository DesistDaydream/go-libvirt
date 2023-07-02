package domain

import (
	"github.com/spf13/cobra"
)

// type DomainFlags struct {
// }

// var domainFlags DomainFlags

func CreateCommand() *cobra.Command {
	domainCmd := &cobra.Command{
		Use:   "domain",
		Short: "虚拟机管理",
		// PersistentPreRun: productsPersistentPreRun,
	}

	domainCmd.AddCommand(
		ListCommand(),
	)

	return domainCmd
}
