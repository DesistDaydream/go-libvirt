# vm-control

基于 libvirt 的 KVM/QEMU 虚拟机批量管理命令行工具。

一次连接多台宿主机，按名称、glob 或状态选择虚拟机，批量执行开机、关机、
内部快照的创建/查看/删除/回滚，并逐台汇总结果。连接只建立一次，命令执行
期间保持有效。

## 依赖

- Go 1.26+
- libvirt 开发库与 pkg-config（CGO 绑定）

```bash
sudo apt install libvirt-dev pkg-config
```

## 构建

```bash
go build -o vm-control ./cmd/vm_control
```

## 连接宿主机

默认通过 `qemu+tcp://<ip>/system` 连接远端 libvirtd。远端需要：

1. `/etc/libvirt/libvirtd.conf` 中设置 `auth_tcp="none"`；
2. 通过 socket 启用 TCP 监听：

```bash
systemctl stop libvirtd.service
systemctl enable libvirtd-tcp.socket --now
systemctl start libvirtd.service
```

3. 防火墙放行 `16509/tcp`。

需要换端口或传输方式时用 `--uri` 覆盖（也可写在配置的 `uri` 字段）：

```bash
# TCP 换端口
./vm-control --ips 10.10.22.62 --uri 'qemu+tcp://{host}:16509/system' domain list

# SSH（需免密登录；known_hosts 无记录时临时加 ?no_verify=1）
./vm-control --ips 10.10.22.62 --uri 'qemu+ssh://root@{host}/system' domain list
```

## 配置

默认读取当前目录的 `my_config.yaml`：

```yaml
ips:
  - 10.10.22.62
domain:
  domains:
    - hdf
```

命令行参数优先于配置文件。

## 使用

### 列出虚拟机

```bash
./vm-control --ips 10.10.22.62 domain list
```

### 创建虚拟机

`domain create` 从 `domain-list.yaml` 读取配置，按名称创建一台虚拟机：

```bash
./vm-control --ips 10.10.22.62 domain create -f domain-list.yaml hdf-01
```

- 根盘来源三选一：`domain_image` 完整拷贝到 `<name>.qcow2`；`base_image` 创建 overlay；`iso` 创建空盘并从 ISO 启动安装；
- 生成 XML 时自动补 `<memballoon model='virtio'/>`，否则 `currentMemory` 无法低于 `memory`；
- 再用基础 XML 生成新 XML，替换名称、CPU、内存、磁盘路径、网桥/MAC；配置 `ip` 时生成 cloud-init seed 并挂成 cdrom；
- 最后定义并启动；`--no-start` 只定义，`--dry-run` 只打印生成的 XML。

字段：`name`、`host`（可选，多宿主机时必填）、`vcpu`、`max_vcpu`、`memory`、`max_memory`、`root_volume`、`volumes`、`ip`、`gateway`、`dns`、`mgmt_nic_name`、`mgmt_nic_bridge`、`macvtap_passthrough_nics`、`pci_passthrough_nics`（可选）、`xml`、`domain_image`/`base_image`/`iso`（三选一）、`pool`（可选，默认 `default`）。

`volumes` 是额外的挂载路径列表，例如 `/data`。每项可以只写路径，也可以写 `{path: /data, size: 50GiB}`；不写 `size` 时默认等于 `root_volume`。文件名是 `<name>-<路径>`（路径中的 `/` 替换为 `-`），按顺序挂为 `vdb`、`vdc`…。libvirt 的块设备 XML 没有“挂载点”字段，guest 内仍需通过 fstab/cloud-init 挂载。

`pci_passthrough_nics` 是宿主机网卡的 PCI 地址列表（如 `0000:3b:00.0`、`0000:3b:00.1`），每项会以 `<hostdev>` 整卡直通给虚拟机，供 guest 内 DPDK 绑定；管理网卡仍走 `mgmt_nic_bridge`。创建前会检查每个设备的 nodedev XML 是否有 `iommuGroup`，没有会直接失败，不会先创建磁盘。

`macvtap_passthrough_nics` 是宿主机镜像口名称列表（如 `enp189s0f0`、`enp189s0f1`），每个口创建一块 virtio 网卡，以 macvtap passthrough 模式接入，供 guest 内 DPDK 分别绑定或做 bond；管理网卡仍走 `mgmt_nic_bridge`。它与 `pci_passthrough_nics` 互斥。这种方式不需要 IOMMU，但同一物理口只能给一台 VM，lowerdev 的 MAC 会变成 VM 的 MAC，性能低于 VFIO 直通。

`domain_image`、`base_image`、`iso` 必须三选一：`domain_image` 先把镜像完整拷贝成 `<name>.qcow2`，用于导入现成系统；`base_image` 只创建 overlay，基础镜像保持只读；`iso` 创建空盘并从 ISO 启动安装。

获取宿主机网卡名称与 PCI 地址：

```bash
./vm-control --ips 10.10.22.62 host nic list
```

输出宿主机、网卡名和 PCI 地址，直接填到 `pci_passthrough_nics`。底层使用的是 libvirt 的 node device 信息；临时排查也可以用 `virsh nodedev-list --cap net` 后跟 `virsh nodedev-dumpxml <device>`。

`ip` 是静态 IPv4 CIDR（如 `10.10.22.180/23`），配置后必须同时给 `gateway`，可选 `dns`；`mgmt_nic_name` 是 guest 内管理网卡名，默认 `enp1s0`，如果基础镜像里不是这个名字必须显式指定，否则 cloud-init 会写 `ifcfg-<名字>`，对不上实际网卡。创建时会生成一个 label 为 `CIDATA` 的 cloud-init NoCloud seed ISO，上传到存储池的 `<name>-seed.iso`，并以 SCSI 光驱挂给虚拟机；`network-config` 按生成的管理网卡 MAC 匹配，设置静态地址，不依赖 DHCP。基础镜像里必须已安装并启用 cloud-init（datasource 含 NoCloud，且未禁用 network config），并且不要再写死静态网卡配置；cloud-init 还会按镜像里的 `disable_root`/`ssh_pwauth` 处理 root SSH，需要 root 密码登录时把这两项改成 `false`/`true`。基础镜像没有 seed 启动时，cloud-init 的 DataSourceNone 会把网络改成 DHCP，不要把基础镜像原来的静态 IP 当成长期配置；`iso` 安装模式不能同时配置 `ip`。overlay 的 `root_volume` 不能小于基础镜像的虚拟大小。

### 批量开关机

```bash
# 开启所有已关机的虚拟机
./vm-control --ips 10.10.22.62 domain start --all --stopped

# 关闭名称匹配 web-* 的虚拟机，并等待完成
./vm-control --ips 10.10.22.62 domain shutdown -d 'web-*' --wait

# 强制关闭所有运行中的虚拟机
./vm-control --ips 10.10.22.62 domain shutdown --all --running --force
```

### 快照

```bash
# 给 hdf 创建名为 before-upgrade 的内部快照
./vm-control --ips 10.10.22.62 domain snapshot create -d hdf -s before-upgrade

# 给所有运行中的虚拟机创建同名快照
./vm-control --ips 10.10.22.62 domain snapshot create --all --running -s daily

# 列出快照（不指定 -d 时列出全部）
./vm-control --ips 10.10.22.62 domain snapshot list

# 回滚
./vm-control --ips 10.10.22.62 domain snapshot revert -d hdf -s before-upgrade

# 删除；默认不删子快照，需要时加 --children
./vm-control --ips 10.10.22.62 domain snapshot delete -d hdf -s before-upgrade
```

`create`、`delete`、`revert` 必须用 `-s` 指定快照名称。

### 选择器

- `-d, --domains`：名称或 glob，逗号分隔，例如 `web-*,db-01`
- `--all`：所有虚拟机
- `--running` / `--stopped`：按状态过滤
- `--parallel`：多宿主机并行，默认开启；同一宿主机内串行

### 快照说明

- 只创建内部快照，要求磁盘为 qcow2；
- `--memory` 同时保存内存状态，`--quiesce` 要求 guest agent 冻结文件系统；
- 快照不是备份，外部快照链未支持；
- 删除快照默认不删除子快照，避免误删链上的其他快照。

### 退出码

- `0`：全部操作成功；
- `1`：至少一台虚拟机失败。结果表会逐台列出成功/失败原因。

## 目录

```text
cmd/vm_control/          CLI 入口、命令与参数
pkg/cloudinit/          cloud-init seed 配置与 ISO 生成
pkg/conf/               my_config 与 domain-list 的解析
pkg/virt/               连接池、虚拟机选择、批量执行、快照、创建
pkg/report/             批量结果输出
```
