package domain

import (
	"fmt"
	"os"
	"strconv"

	"github.com/DesistDaydream/go-libvirt/pkg/handler"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"libvirt.org/go/libvirt"
)

func ListCommand() *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "列出所有虚拟机",
		Run:   runList,
	}

	return listCmd
}

func runList(cmd *cobra.Command, args []string) {
	for _, conn := range handler.Conns {
		defer conn.Close()

		var (
			allCPU int
			allMem float64
		)

		curHost, _ := conn.GetHostname()
		fmt.Printf("======== %v ========\n", curHost)

		table := tablewriter.NewWriter(os.Stdout)

		table.SetHeader([]string{"实例", "状态", "CPU", "CPU使用率", "内存", "内存使用率"})

		conn.ListDomains()

		doms, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
		if err != nil {
			logrus.Errorf("无法列出 Domain，原因: %v", err)
		}

		for _, dom := range doms {
			var domStat string

			domName, _ := dom.GetName()

			domInfo, _ := dom.GetInfo()
			if domInfo.State == 1 {
				domStat = fmt.Sprintf("\033[0;37;42m%s\033[0m", "开机")
			} else {
				domStat = fmt.Sprintf("\033[0;37;41m%s\033[0m", "未开机")
			}

			// cpuUsage, _ := domainCpuUsage(dom)

			table.Append([]string{domName, domStat, strconv.Itoa(int(domInfo.NrVirtCpu)), "", fmt.Sprintf("%.2f GiB", float64(domInfo.Memory)/1024/1024), ""})

			allCPU = allCPU + int(domInfo.NrVirtCpu)
			allMem = allMem + float64(domInfo.Memory)

			dom.Free()
		}

		table.Render()

		nodeInfo, _ := conn.GetNodeInfo()

		fmt.Printf("当前服务器共有 %v CPU，已分配给虚拟机 %v CPU\n", nodeInfo.Cpus, allCPU)
		fmt.Printf("当前服务器共有 %.2f GiB 内存，已分配给虚拟机 %.2f GiB 内存\n", float64(nodeInfo.Memory/1024/1024), float64(allMem/1024/1024))
	}

}
