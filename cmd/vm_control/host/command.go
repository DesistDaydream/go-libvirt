package host

import (
	"github.com/spf13/cobra"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/app"
)

func CreateCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "host",
		Short:             "宿主机管理",
		PersistentPreRunE: a.Init,
	}

	cmd.AddCommand(nicCommand(a))
	return cmd
}
