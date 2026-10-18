package virt

import (
	"sync"
)

// Action 对单台虚拟机执行一次操作。note 用于补充说明（例如「已在运行」）。
type Action func(*Domain) (note string, err error)

// Result 是一台虚拟机的执行结果。
type Result struct {
	Host   string
	Domain string
	Action string
	Note   string
	Err    error
}

// Run 对选中的虚拟机执行 action。
//
// libvirt 连接不是线程安全的，因此并发只发生在宿主机之间：每台宿主机
// 由自己的 goroutine 串行处理，绝不共享连接。action 返回后句柄立即释放。
func Run(domains []*Domain, action string, fn Action, parallel bool) []Result {
	groups := make(map[*Host][]*Domain)
	var hosts []*Host
	for _, d := range domains {
		if _, ok := groups[d.Host]; !ok {
			hosts = append(hosts, d.Host)
		}
		groups[d.Host] = append(groups[d.Host], d)
	}

	runHost := func(h *Host) []Result {
		results := make([]Result, 0, len(groups[h]))
		for _, d := range groups[h] {
			note, err := fn(d)
			d.Free()
			results = append(results, Result{
				Host:   h.Name,
				Domain: d.Name,
				Action: action,
				Note:   note,
				Err:    err,
			})
		}
		return results
	}

	if !parallel || len(hosts) <= 1 {
		var results []Result
		for _, h := range hosts {
			results = append(results, runHost(h)...)
		}
		return results
	}

	perHost := make([][]Result, len(hosts))
	var wg sync.WaitGroup
	for i, h := range hosts {
		wg.Add(1)
		go func(i int, h *Host) {
			defer wg.Done()
			perHost[i] = runHost(h)
		}(i, h)
	}
	wg.Wait()

	var results []Result
	for _, hostResults := range perHost {
		results = append(results, hostResults...)
	}
	return results
}
