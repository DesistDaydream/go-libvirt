package report

import (
	"fmt"
	"io"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"

	"github.com/DesistDaydream/go-libvirt/pkg/virt"
)

// Results 输出批量执行结果，返回失败数量。
func Results(w io.Writer, results []virt.Result) (int, error) {
	if len(results) == 0 {
		return 0, nil
	}

	table := tablewriter.NewTable(w,
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithRowMaxWidth(80),
	)
	table.Header("宿主机", "虚拟机", "操作", "结果")

	failures := 0
	for _, result := range results {
		host := result.Host
		if host == "" {
			host = "-"
		}

		var status string
		switch {
		case result.Err != nil:
			failures++
			status = "失败: " + result.Err.Error()
		case result.Note != "":
			status = "成功（" + result.Note + "）"
		default:
			status = "成功"
		}

		if err := table.Append([]string{host, result.Domain, result.Action, status}); err != nil {
			return 0, err
		}
	}

	if err := table.Render(); err != nil {
		return 0, err
	}
	fmt.Fprintf(w, "共 %d 个操作：成功 %d，失败 %d\n", len(results), len(results)-failures, failures)
	return failures, nil
}
