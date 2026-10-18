package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"libvirt.org/go/libvirt"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/app"
	"github.com/DesistDaydream/go-libvirt/pkg/virt"
)

func shutdownCommand(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "shutdown",
		Aliases: []string{"stop"},
		Short:   "关闭虚拟机",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runShutdown(a)
		},
	}

	pf := cmd.Flags()
	pf.BoolVar(&a.Flags.ShutdownWait, "wait", false, "等待虚拟机关机完成")
	pf.BoolVar(&a.Flags.ShutdownForce, "force", false, "强制关机（Destroy），不发送 ACPI 关机信号")
	pf.DurationVar(&a.Flags.ShutdownTimeout, "timeout", 5*time.Minute, "等待关机的超时时间")

	return cmd
}

func runShutdown(a *app.App) error {
	selection, err := a.Select(true)
	if err != nil {
		return err
	}
	if a.Flags.ShutdownWait && a.Flags.ShutdownTimeout <= 0 {
		return errors.New("--timeout 必须大于 0")
	}

	return a.RunBatch("shutdown", selection, func(d *virt.Domain) (string, error) {
		state, _, err := d.Vir.GetState()
		if err != nil {
			return "", err
		}
		if state == libvirt.DOMAIN_SHUTOFF || state == libvirt.DOMAIN_CRASHED {
			return "已关机", nil
		}

		if a.Flags.ShutdownForce {
			err = d.Vir.Destroy()
		} else {
			err = d.Vir.Shutdown()
		}
		if err != nil {
			return "", err
		}
		if !a.Flags.ShutdownWait {
			return "", nil
		}
		return "", waitShutdown(d.Vir, a.Flags.ShutdownTimeout)
	})
}

func waitShutdown(dom *libvirt.Domain, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		state, _, err := dom.GetState()
		if err != nil {
			return err
		}
		if state == libvirt.DOMAIN_SHUTOFF {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("等待关机超时（%s）", timeout)
		}
		time.Sleep(2 * time.Second)
	}
}
