package main

import (
	"time"

	"github.com/sirupsen/logrus"
	"libvirt.org/go/libvirt"

	humanize "github.com/dustin/go-humanize"
)

func main() {
	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	doms, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE)
	if err != nil {
		panic(err)
	}

	logrus.Info("%d running domains:\n", len(doms))

	for _, dom := range doms {
		d, err := dom.GetInfo()
		if err != nil {
			logrus.Error("获取虚拟机信息异常: ", err)
		}

		name, err := dom.GetName()
		if err != nil {
			logrus.Error("获取虚拟机名称异常: ", err)
		}

		logrus.WithFields(logrus.Fields{
			"状态":      d.State,
			"最大内存":    humanize.IBytes(d.MaxMem),
			"已用内存":    humanize.IBytes(d.Memory),
			"逻辑CPU数":  d.NrVirtCpu,
			"CPU运行时间": time.Nanosecond * time.Duration(d.CpuTime),
		}).Infof("%v 虚拟机信息", name)

		dom.Free()
	}
}
