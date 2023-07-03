package main

import (
	"os"

	logging "github.com/DesistDaydream/logging/pkg/logrus_init"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/domain"
	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/flags"
	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/snapshot"
	"github.com/DesistDaydream/go-libvirt/pkg/handler"
)

func main() {
	app := newApp()
	err := app.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func newApp() *cobra.Command {
	long := ``

	var RootCmd = &cobra.Command{
		Use:   "vm-control",
		Short: "KVM/QEMU 虚拟机管理",
		Long:  long,
		// PersistentPreRun: rootPersistentPreRun,
	}

	cobra.OnInitialize(initConfig)

	logging.AddFlags(&flags.L)
	RootCmd.PersistentFlags().StringSliceVar(&flags.F.Hosts, "hosts", nil, "宿主机列表")
	// RootCmd.PersistentFlags().StringVar(&flags.ConfigPath, "config-path", "", "配置文件路径")
	// RootCmd.PersistentFlags().StringVar(&flags.ConfigName, "config-name", "", "配置文件名称")

	// 添加子命令
	RootCmd.AddCommand(
		domain.CreateCommand(),
		snapshot.CreateCommand(),
	)

	return RootCmd
}

// 执行每个 root 下的子命令时，都需要执行的函数
func initConfig() {
	if err := logging.LogrusInit(&flags.L); err != nil {
		logrus.Fatal("初始化日志失败", err)
	}

	handler.NewHandler()
}
