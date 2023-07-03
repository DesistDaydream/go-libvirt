package domain

import (
	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/flags"
	"github.com/DesistDaydream/go-libvirt/pkg/handler"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func shutdownCommand() *cobra.Command {
	var shutdownCmd = &cobra.Command{
		Use:   "shutdown",
		Short: "关闭指定虚拟机",
		Run:   runShutdown,
	}

	return shutdownCmd
}

func runShutdown(cmd *cobra.Command, args []string) {
	if flags.DF.DomainsName == nil {
		logrus.Fatal("请指定虚拟机名称")
	}

	domains := handler.FindNeedHandleDomains(flags.DF.DomainsName)
	for _, d := range domains {
		dName, _ := d.VirDomain.GetName()
		logrus.Debugf("开始处理 %v", dName)

		err := d.VirDomain.Shutdown()
		if err != nil {
			logrus.Errorf("关机失败，原因: %v", err)
		}
	}
}
