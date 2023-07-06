package handler

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"libvirt.org/go/libvirt"
)

var Conns []*libvirt.Connect

func NewLibvirtConnect(ips []string) {
	for _, host := range ips {
		conn, err := libvirt.NewConnect(fmt.Sprintf("qemu+tcp://%s/system", host))
		if err != nil {
			logrus.Fatalf("无法连接到 %v，原因: %v", host, err)
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
