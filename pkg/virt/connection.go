package virt

import (
	"errors"
	"fmt"
	"strings"

	"libvirt.org/go/libvirt"
)

// DefaultURI 是未显式指定 --uri 时使用的连接模板，默认走 TCP。
const DefaultURI = "qemu+tcp://{host}/system"

// Host 是一条已建立的 libvirt 连接。
type Host struct {
	// Name 用于输出，优先使用 libvirt 返回的主机名。
	Name string
	// Addr 是 --ips 中对应的地址；固定 URI 连接时为空。
	Addr string
	Conn *libvirt.Connect
}

// Pool 持有本次执行期间所有宿主机连接。
//
// Pool 的生命周期就是进程的生命周期：命令执行完由 main 统一关闭，
// 这样命令拿到的 Domain 句柄在整条命令期间始终有效。
type Pool struct {
	hosts []*Host
}

// Dial 按 URI 模板连接所有宿主机。模板中的 {host} 会替换成 --ips 的每个地址。
func Dial(ips []string, uriTemplate string) (*Pool, error) {
	uris, addrs, err := buildURIs(ips, uriTemplate)
	if err != nil {
		return nil, err
	}

	pool := &Pool{}
	var failures []error
	for i, uri := range uris {
		conn, err := libvirt.NewConnect(uri)
		if err != nil {
			failures = append(failures, fmt.Errorf("连接 %s 失败: %w", displayAddr(addrs[i], uri), err))
			continue
		}

		host := &Host{Addr: addrs[i], Conn: conn}
		host.Name = host.Addr
		if hostname, err := conn.GetHostname(); err == nil && hostname != "" {
			host.Name = hostname
		}
		pool.hosts = append(pool.hosts, host)
	}

	if len(failures) > 0 {
		pool.Close()
		return nil, errors.Join(failures...)
	}
	if len(pool.hosts) == 0 {
		return nil, errors.New("没有可连接的 libvirt 宿主机")
	}
	return pool, nil
}

// buildURIs 把 --ips 与 --uri 模板展开成实际连接地址。
// 模板可以是任意 libvirt URI（qemu+tcp、qemu+ssh、qemu+tls 等），
// 默认使用 DefaultURI（TCP）。
func buildURIs(ips []string, uriTemplate string) (uris []string, addrs []string, err error) {
	uriTemplate = strings.TrimSpace(uriTemplate)
	if uriTemplate == "" {
		uriTemplate = DefaultURI
	}

	if strings.Contains(uriTemplate, "{host}") {
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip == "" {
				continue
			}
			uris = append(uris, strings.ReplaceAll(uriTemplate, "{host}", ip))
			addrs = append(addrs, ip)
		}
		if len(uris) == 0 {
			return nil, nil, errors.New("URI 模板包含 {host}，但没有指定宿主机；请用 --ips 指定")
		}
		return uris, addrs, nil
	}

	// 固定 URI 时地址没有意义；以 --uri 为准。
	return []string{uriTemplate}, []string{""}, nil
}

func displayAddr(addr, uri string) string {
	if addr != "" {
		return addr
	}
	return uri
}

func (p *Pool) Hosts() []*Host {
	return p.hosts
}

func (p *Pool) Close() {
	for _, host := range p.hosts {
		_, _ = host.Conn.Close()
	}
	p.hosts = nil
}
