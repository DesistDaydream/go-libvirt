package handler

import (
	"fmt"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/flags"
	"libvirt.org/go/libvirt"
)

var Conns []*libvirt.Connect

func NewHandler() {
	for _, hosts := range flags.F.Hosts {
		conn, err := libvirt.NewConnect(fmt.Sprintf("qemu+tcp://%s/system", hosts))
		if err != nil {
			panic(err)
		}
		Conns = append(Conns, conn)
	}
}
