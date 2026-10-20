# 创建虚拟机流程

`vm-control domain create -f domain-list.yaml <name>` 的步骤。入口 `cmd/vm_control/domain/create.go`，主流程 `pkg/virt/create.go` 与 `pkg/virt/domainxml.go`。

## 一、连接

不产生任何资源，只建立会话。

1. 读 `domain-list.yaml`，按名称取出域定义，用 `qemu+tcp://<ip>/system` 连接宿主机。

## 二、修改

改内存里的基础 XML。

2. 读基础 XML（`xml` 字段），把它的 `<name>` 记为 baseName 后改成目标名，删掉 `<uuid>`。
3. `os/nvram` 文本里把 baseName 换成目标名；不含 baseName 时改成同目录的 `<name>_VARS.fd`。
4. 改写 `memory`/`currentMemory`、`vcpu`/`vcpu current`。
5. 磁盘改名（文件名替换 baseName）；第一块盘按来源选整盘拷贝（`domain_image`）或 overlay（`base_image`），`volumes` 生成 `<name>-<挂载路径>.qcow2`。
6. 生成随机 MAC；配置 `mgmt_nic_bridge` 时换成 bridge 接口；按配置追加 macvtap 网卡与 PCI hostdev。
7. 补 `memballoon`；`iso` 模式挂光盘并把启动顺序设为 `cdrom` → `hd`。

## 三、创建

在宿主机上真正产生资源

8. 建卷：整盘拷贝、新建空卷或 overlay。
9. 配置了 `ip` 时写入对应 libvirt 网络的静态 DHCP。
10. `DomainDefineXML` 定义域。
11. `domain.Create()` 启动域，libvirt 此时从 `/usr/share/edk2/aarch64/vars-template-pflash.raw` 复制出 `/var/lib/libvirt/qemu/nvram/<name>_VARS.fd`；`--no-start` 跳过启动。

## 四、检查

12. 打印磁盘路径、MAC、IP、网桥（回显计划值，不读回域状态）。
