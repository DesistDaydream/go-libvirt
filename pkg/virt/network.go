package virt

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"libvirt.org/go/libvirt"
)

var errNetworkNotFound = errors.New("未找到匹配的 libvirt 网络")

type dhcpReservation struct {
	network *libvirt.Network
	xml     string
	flags   libvirt.NetworkUpdateFlags
}

// reserveDHCP 在网桥对应的 libvirt 网络中写入静态 DHCP 记录。
// 找不到对应网络时仅告警，不影响虚拟机创建。
func reserveDHCP(host *Host, plan *createPlan, spec CreateSpec) (*dhcpReservation, error) {
	if spec.IP == "" {
		return nil, nil
	}

	network, err := findNetwork(host, plan.MgmtNICBridge, plan.NetworkName)
	if errors.Is(err, errNetworkNotFound) {
		logrus.Warnf("未找到与 mgmt_nic_bridge %q 对应的 libvirt 网络，IP %s 未做静态 DHCP 绑定", plan.MgmtNICBridge, spec.IP)
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查找 libvirt 网络失败: %w", err)
	}

	hostXML := fmt.Sprintf("<host mac='%s' name='%s' ip='%s'/>", plan.MAC, escapeXML(spec.Name), spec.IP)
	flags := libvirt.NETWORK_UPDATE_AFFECT_CONFIG
	if active, err := network.IsActive(); err == nil && active {
		flags |= libvirt.NETWORK_UPDATE_AFFECT_LIVE
	}

	if err := network.Update(libvirt.NETWORK_UPDATE_COMMAND_ADD_LAST, libvirt.NETWORK_SECTION_IP_DHCP_HOST, -1, hostXML, flags); err != nil {
		_ = network.Free()
		return nil, fmt.Errorf("添加静态 DHCP 记录失败: %w", err)
	}

	return &dhcpReservation{network: network, xml: hostXML, flags: flags}, nil
}

func findNetwork(host *Host, bridge, networkName string) (*libvirt.Network, error) {
	if bridge != "" {
		networks, err := host.Conn.ListAllNetworks(libvirt.CONNECT_LIST_NETWORKS_ACTIVE | libvirt.CONNECT_LIST_NETWORKS_INACTIVE)
		if err != nil {
			return nil, err
		}

		index := -1
		for i := range networks {
			name, err := networks[i].GetBridgeName()
			if err == nil && name == bridge {
				index = i
				break
			}
		}
		for i := range networks {
			if i != index {
				_ = networks[i].Free()
			}
		}
		if index == -1 {
			return nil, errNetworkNotFound
		}
		return &networks[index], nil
	}

	if networkName != "" {
		network, err := host.Conn.LookupNetworkByName(networkName)
		if err != nil {
			return nil, errNetworkNotFound
		}
		return network, nil
	}

	return nil, errNetworkNotFound
}

func (r *dhcpReservation) remove() {
	if r == nil {
		return
	}
	if err := r.network.Update(libvirt.NETWORK_UPDATE_COMMAND_DELETE, libvirt.NETWORK_SECTION_IP_DHCP_HOST, -1, r.xml, r.flags); err != nil {
		logrus.Warnf("回滚静态 DHCP 记录失败: %v", err)
	}
	_ = r.network.Free()
}
