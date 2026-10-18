package virt

import (
	"errors"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
	"libvirt.org/go/libvirt"
)

// CreateSpec 是创建虚拟机所需的全部输入。
type CreateSpec struct {
	Name                   string
	VCPU                   int
	MaxVCPU                int
	Memory                 uint64 // bytes
	MaxMemory              uint64 // bytes
	RootVolume             uint64 // bytes
	Volumes                []CreateVolume
	IP                     string
	MgmtNICBridge          string
	MacvtapPassthroughNICs []string
	PCIPassthroughNICs     []string
	BaseImage              string
	DomainImage            string
	ISO                    string
	Pool                   string
	XMLPath                string
}

// CreateVolume 是额外数据盘；Size 为 0 时使用 RootVolume。
type CreateVolume struct {
	Path string
	Size uint64
}

type CreateOptions struct {
	Start  bool
	DryRun bool
	Out    io.Writer
}

// CreateDomain 依据基础 XML 和 CreateSpec 创建磁盘、定义并启动虚拟机。
func CreateDomain(host *Host, spec CreateSpec, opts CreateOptions) error {
	if opts.Out == nil {
		opts.Out = io.Discard
	}

	plan, err := buildCreatePlan(spec)
	if err != nil {
		return err
	}
	if opts.DryRun {
		return printCreatePlan(opts.Out, plan)
	}

	if len(spec.MacvtapPassthroughNICs) > 0 && len(spec.PCIPassthroughNICs) > 0 {
		return errors.New("macvtap_passthrough_nics 与 pci_passthrough_nics 不能同时使用")
	}
	if err := validatePCIPassthroughNICs(host, spec.PCIPassthroughNICs); err != nil {
		return err
	}
	if err := validateMacvtapPassthroughNICs(host, spec.MacvtapPassthroughNICs); err != nil {
		return err
	}

	volumes, err := createVolumes(host, spec, plan.Disks)
	if err != nil {
		return err
	}

	reservation, err := reserveDHCP(host, plan, spec)
	if err != nil {
		deleteVolumes(volumes)
		return err
	}

	domain, err := host.Conn.DomainDefineXML(plan.XML)
	if err != nil {
		deleteVolumes(volumes)
		reservation.remove()
		return fmt.Errorf("定义虚拟机失败: %w", err)
	}
	defer domain.Free()
	freeVolumes(volumes)

	if opts.Start {
		if err := domain.Create(); err != nil {
			return fmt.Errorf("虚拟机 %s 已定义，但启动失败: %w", spec.Name, err)
		}
	}

	fmt.Fprintf(opts.Out, "已创建虚拟机 %s（%s）\n", spec.Name, host.Name)
	for _, disk := range plan.Disks {
		fmt.Fprintf(opts.Out, "  磁盘: %s\n", disk.Path)
	}
	fmt.Fprintf(opts.Out, "  MAC: %s\n", plan.MAC)
	if spec.IP != "" {
		fmt.Fprintf(opts.Out, "  IP: %s\n", spec.IP)
	}
	if spec.MgmtNICBridge != "" {
		fmt.Fprintf(opts.Out, "  网桥: %s\n", spec.MgmtNICBridge)
	}
	return nil
}

func printCreatePlan(w io.Writer, plan *createPlan) error {
	for _, disk := range plan.Disks {
		fmt.Fprintf(w, "===== 磁盘 %s =====\n%s\n\n", disk.Path, volumeXML(disk))
	}
	fmt.Fprintf(w, "===== 虚拟机 XML =====\n%s\n", plan.XML)
	return nil
}

// deleteVolumes 用于创建失败时的回滚。
func deleteVolumes(volumes []*libvirt.StorageVol) {
	for _, volume := range volumes {
		if err := volume.Delete(libvirt.StorageVolDeleteFlags(0)); err != nil {
			logrus.Warnf("回滚磁盘失败: %v", err)
		}
		_ = volume.Free()
	}
}

func freeVolumes(volumes []*libvirt.StorageVol) {
	for _, volume := range volumes {
		_ = volume.Free()
	}
}
