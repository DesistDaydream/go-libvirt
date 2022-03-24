package main

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"libvirt.org/go/libvirt"
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

	fmt.Printf("%d running domains:\n", len(doms))
	for _, dom := range doms {
		d, err := dom.GetInfo()
		if err == nil {
			logrus.Error(d)
		}
		logrus.Info(d)
		dom.Free()
	}
}
