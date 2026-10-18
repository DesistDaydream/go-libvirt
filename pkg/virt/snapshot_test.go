package virt

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestSnapshotXMLString(t *testing.T) {
	opts := SnapshotOptions{
		Name:        `snap&<1>`,
		Description: `描述 > "引号"`,
		Memory:      false,
	}

	doc, err := snapshotXMLString(opts, []string{"vda", "vdb"})
	if err != nil {
		t.Fatalf("snapshotXMLString 返回错误: %v", err)
	}

	if strings.Contains(doc, "snap&<1>") {
		t.Errorf("快照名未转义: %s", doc)
	}
	if !strings.Contains(doc, "snap&amp;&lt;1&gt;") {
		t.Errorf("快照名转义结果不符合预期: %s", doc)
	}

	var got snapshotXML
	if err := xml.Unmarshal([]byte(doc), &got); err != nil {
		t.Fatalf("生成的 XML 无法解析: %v\n%s", err, doc)
	}
	if got.Name != opts.Name {
		t.Errorf("name = %q，期望 %q", got.Name, opts.Name)
	}
	if got.Description != opts.Description {
		t.Errorf("description = %q，期望 %q", got.Description, opts.Description)
	}
	if got.Memory == nil || got.Memory.Snapshot != "no" {
		t.Errorf("memory = %+v，期望 snapshot=no", got.Memory)
	}
	if got.Disks == nil || len(got.Disks.Disks) != 2 {
		t.Fatalf("disks = %+v，期望 2 块磁盘", got.Disks)
	}
	if got.Disks.Disks[0].Name != "vda" || got.Disks.Disks[0].Snapshot != "internal" {
		t.Errorf("第一块磁盘 = %+v", got.Disks.Disks[0])
	}
}

func TestSnapshotXMLStringMemory(t *testing.T) {
	doc, err := snapshotXMLString(SnapshotOptions{Name: "s1", Memory: true}, nil)
	if err != nil {
		t.Fatalf("snapshotXMLString 返回错误: %v", err)
	}

	var got snapshotXML
	if err := xml.Unmarshal([]byte(doc), &got); err != nil {
		t.Fatalf("生成的 XML 无法解析: %v\n%s", err, doc)
	}
	if got.Memory == nil || got.Memory.Snapshot != "internal" {
		t.Errorf("memory = %+v，期望 snapshot=internal", got.Memory)
	}
	if got.Disks != nil {
		t.Errorf("disks = %+v，期望为空", got.Disks)
	}
}
