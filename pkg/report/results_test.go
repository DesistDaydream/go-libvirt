package report

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/DesistDaydream/go-libvirt/pkg/virt"
)

func TestResults(t *testing.T) {
	results := []virt.Result{
		{Host: "host-1", Domain: "web-01", Action: "start"},
		{Host: "host-1", Domain: "web-02", Action: "start", Err: errors.New("domain is already running")},
		{Host: "host-2", Domain: "db-01", Action: "start", Note: "已在运行"},
	}

	var buf bytes.Buffer
	failures, err := Results(&buf, results)
	if err != nil {
		t.Fatalf("Results 返回错误: %v", err)
	}
	if failures != 1 {
		t.Errorf("failures = %d，期望 1", failures)
	}

	out := buf.String()
	for _, want := range []string{"web-01", "web-02", "db-01", "domain is already running", "成功（已在运行）", "共 3 个操作：成功 2，失败 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("输出缺少 %q:\n%s", want, out)
		}
	}
}

func TestResultsEmpty(t *testing.T) {
	var buf bytes.Buffer
	failures, err := Results(&buf, nil)
	if err != nil {
		t.Fatalf("Results 返回错误: %v", err)
	}
	if failures != 0 || buf.Len() != 0 {
		t.Errorf("空结果不应输出内容: failures=%d output=%q", failures, buf.String())
	}
}
