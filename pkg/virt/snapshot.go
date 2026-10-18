package virt

import (
	"encoding/xml"
	"fmt"
	"time"

	"libvirt.org/go/libvirt"
)

// SnapshotOptions 描述一次内部快照创建。
type SnapshotOptions struct {
	Name        string
	Description string
	// Memory 为 true 时同时保存内存状态；只对运行中的虚拟机有意义。
	Memory bool
	// Quiesce 要求 guest agent 冻结文件系统，需要虚拟机内已安装并运行 qemu-guest-agent。
	Quiesce bool
}

// Snapshot 是快照列表中的一行。
type Snapshot struct {
	Name         string
	Description  string
	CreationTime time.Time
	Current      bool
}

type snapshotXML struct {
	XMLName     xml.Name          `xml:"domainsnapshot"`
	Name        string            `xml:"name"`
	Description string            `xml:"description,omitempty"`
	Memory      *snapshotMemory   `xml:"memory,omitempty"`
	Disks       *snapshotDisksXML `xml:"disks,omitempty"`
}

type snapshotMemory struct {
	Snapshot string `xml:"snapshot,attr"`
}

type snapshotDisksXML struct {
	Disks []snapshotDisk `xml:"disk"`
}

type snapshotDisk struct {
	Name     string `xml:"name,attr"`
	Snapshot string `xml:"snapshot,attr"`
}

type snapshotDescription struct {
	Name         string `xml:"name"`
	Description  string `xml:"description"`
	CreationTime int64  `xml:"creationTime"`
}

type domainDisksXML struct {
	Devices struct {
		Disks []struct {
			Device string `xml:"device,attr"`
			Target struct {
				Dev string `xml:"dev,attr"`
			} `xml:"target"`
		} `xml:"disk"`
	} `xml:"devices"`
}

// CreateSnapshot 创建内部快照（qcow2）。快照覆盖所有 device=disk 的磁盘。
func CreateSnapshot(d *Domain, opts SnapshotOptions) (string, error) {
	if opts.Name == "" {
		return "", fmt.Errorf("快照名称不能为空")
	}

	disks, err := domainDiskTargets(d.Vir)
	if err != nil {
		return "", err
	}
	xmlDoc, err := snapshotXMLString(opts, disks)
	if err != nil {
		return "", err
	}

	flags := libvirt.DomainSnapshotCreateFlags(0)
	if opts.Quiesce {
		flags |= libvirt.DOMAIN_SNAPSHOT_CREATE_QUIESCE
	}

	snapshot, err := d.Vir.CreateSnapshotXML(xmlDoc, flags)
	if err != nil {
		return "", err
	}
	defer snapshot.Free()

	return snapshot.GetName()
}

func domainDiskTargets(dom *libvirt.Domain) ([]string, error) {
	desc, err := dom.GetXMLDesc(0)
	if err != nil {
		return nil, fmt.Errorf("读取虚拟机 XML 失败: %w", err)
	}

	var dx domainDisksXML
	if err := xml.Unmarshal([]byte(desc), &dx); err != nil {
		return nil, fmt.Errorf("解析虚拟机 XML 失败: %w", err)
	}

	var targets []string
	for _, disk := range dx.Devices.Disks {
		if disk.Device != "disk" || disk.Target.Dev == "" {
			continue
		}
		targets = append(targets, disk.Target.Dev)
	}
	return targets, nil
}

// snapshotXMLString 用 encoding/xml 生成快照 XML，名称与描述由 xml 包负责转义。
func snapshotXMLString(opts SnapshotOptions, disks []string) (string, error) {
	doc := snapshotXML{
		Name:        opts.Name,
		Description: opts.Description,
		Memory:      &snapshotMemory{Snapshot: "no"},
	}
	if opts.Memory {
		doc.Memory.Snapshot = "internal"
	}
	if len(disks) > 0 {
		doc.Disks = &snapshotDisksXML{}
		for _, disk := range disks {
			doc.Disks.Disks = append(doc.Disks.Disks, snapshotDisk{Name: disk, Snapshot: "internal"})
		}
	}

	body, err := xml.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("生成快照 XML 失败: %w", err)
	}
	return string(body), nil
}

// ListSnapshots 列出虚拟机上的全部快照。
func ListSnapshots(d *Domain) ([]Snapshot, error) {
	snapshots, err := d.Vir.ListAllSnapshots(0)
	if err != nil {
		return nil, err
	}

	out := make([]Snapshot, 0, len(snapshots))
	for i := range snapshots {
		s := &snapshots[i]
		name, err := s.GetName()
		if err != nil {
			_ = s.Free()
			continue
		}

		var desc snapshotDescription
		if xmlDoc, err := s.GetXMLDesc(0); err == nil {
			_ = xml.Unmarshal([]byte(xmlDoc), &desc)
		}
		current, _ := s.IsCurrent(0)

		snapshot := Snapshot{
			Name:        name,
			Description: desc.Description,
			Current:     current,
		}
		if desc.CreationTime > 0 {
			snapshot.CreationTime = time.Unix(desc.CreationTime, 0)
		}
		out = append(out, snapshot)
		_ = s.Free()
	}
	return out, nil
}

// DeleteSnapshot 删除指定快照。children 为 true 时连同子快照一起删除。
func DeleteSnapshot(d *Domain, name string, children bool) error {
	snapshot, err := d.Vir.SnapshotLookupByName(name, 0)
	if err != nil {
		return err
	}
	defer snapshot.Free()

	var flags libvirt.DomainSnapshotDeleteFlags
	if children {
		flags |= libvirt.DOMAIN_SNAPSHOT_DELETE_CHILDREN
	}
	return snapshot.Delete(flags)
}

// RevertSnapshot 回滚到指定快照。
func RevertSnapshot(d *Domain, name string, force, running, paused bool) error {
	snapshot, err := d.Vir.SnapshotLookupByName(name, 0)
	if err != nil {
		return err
	}
	defer snapshot.Free()

	var flags libvirt.DomainSnapshotRevertFlags
	if force {
		flags |= libvirt.DOMAIN_SNAPSHOT_REVERT_FORCE
	}
	if running {
		flags |= libvirt.DOMAIN_SNAPSHOT_REVERT_RUNNING
	}
	if paused {
		flags |= libvirt.DOMAIN_SNAPSHOT_REVERT_PAUSED
	}
	return snapshot.RevertToSnapshot(flags)
}
