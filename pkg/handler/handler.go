package handler

import (
	"fmt"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/flags"
	"github.com/sirupsen/logrus"
	"libvirt.org/go/libvirt"
)

var Conns []*libvirt.Connect

func NewLibvirtConnect() {
	for _, hosts := range flags.F.IPs {
		conn, err := libvirt.NewConnect(fmt.Sprintf("qemu+tcp://%s/system", hosts))
		if err != nil {
			panic(err)
		}
		Conns = append(Conns, conn)
	}
}

type NeedHandleDomain struct {
	HostName   string
	DomainName string
	VirDomain  *libvirt.Domain
}

var Domains []NeedHandleDomain

func FindNeedHandleDomains(DomainName []string) []NeedHandleDomain {
	for _, conn := range Conns {
		defer conn.Close()

		hostname, _ := conn.GetHostname()
		for _, dName := range DomainName {
			domain, _ := conn.LookupDomainByName(dName)
			if domain != nil {
				Domains = append(Domains, NeedHandleDomain{
					HostName:   hostname,
					DomainName: dName,
					VirDomain:  domain,
				})
			}
		}
	}

	for _, d := range Domains {
		logrus.Debugf("在 %v 上找到需要处理的虚拟机: %v", d.HostName, d.DomainName)
	}

	return Domains
}
