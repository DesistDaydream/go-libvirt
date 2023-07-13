package main

import (
	"fmt"
	"os"

	logging "github.com/DesistDaydream/logging/pkg/logrus_init"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/domain"
	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/flags"

	"github.com/DesistDaydream/go-libvirt/pkg/handler"
)

func main() {
	app := newApp()
	err := app.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var rootCmd *cobra.Command

func newApp() *cobra.Command {
	var long string = ``

	rootCmd = &cobra.Command{
		Use:   "vm-control",
		Short: "KVM/QEMU 虚拟机管理",
		Long:  long,
		// PersistentPreRun: rootPersistentPreRun,
	}

	cobra.OnInitialize(initConfig)

	logging.AddFlags(&flags.L)
	rootCmd.PersistentFlags().StringSliceVar(&flags.F.IPs, "ips", nil, "宿主机 IP 列表")
	rootCmd.PersistentFlags().StringVar(&flags.F.ConfigPath, "config-path", ".", "配置文件路径")
	rootCmd.PersistentFlags().StringVar(&flags.F.ConfigName, "config-name", "my_config", "配置文件名称")

	// 添加子命令
	rootCmd.AddCommand(
		domain.CreateCommand(),
	)

	return rootCmd
}

// 执行每个 root 下的子命令时，都需要执行的函数
func initConfig() {
	if err := logging.LogrusInit(&flags.L); err != nil {
		logrus.Fatalf("初始化日志失败: %v", err)
	}

	viper.SetConfigName(flags.F.ConfigName)
	viper.AddConfigPath(flags.F.ConfigPath)
	// viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Failed to read config file:", err)
		return
	}
	viper.BindPFlags(rootCmd.Flags())
	ips := viper.GetStringSlice("ips")

	handler.NewLibvirtConnect(ips)
}
