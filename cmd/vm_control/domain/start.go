package domain

import (
	"github.com/spf13/cobra"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/app"
	"github.com/DesistDaydream/go-libvirt/pkg/virt"
)

func startCommand(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "开启虚拟机",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runStart(a)
		},
	}
}

func runStart(a *app.App) error {
	selection, err := a.Select(true)
	if err != nil {
		return err
	}

	return a.RunBatch("start", selection, func(d *virt.Domain) (string, error) {
		active, err := d.Vir.IsActive()
		if err != nil {
			return "", err
		}
		if active {
			return "已在运行", nil
		}
		return "", d.Vir.Create()
	})
}
