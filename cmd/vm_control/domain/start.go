package domain

import (
	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/flags"
	"github.com/DesistDaydream/go-libvirt/pkg/handler"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func startCommand() *cobra.Command {
	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "开启指定虚拟机",
		Run:   runStart,
	}

	return startCmd
}

func runStart(cmd *cobra.Command, args []string) {
	if flags.DF.DomainsName == nil {
		logrus.Fatal("请指定虚拟机名称")
	}

	domains := handler.FindNeedHandleDomains(flags.DF.DomainsName)
	for _, d := range domains {
		dName, _ := d.VirDomain.GetName()
		logrus.Debugf("开始处理 %v", dName)

		err := d.VirDomain.Create()
		if err != nil {
			logrus.Errorf("%v 开机失败，原因: %v", dName, err)
		} else {
			logrus.Infof("%v 开机成功", dName)
		}
	}
}
