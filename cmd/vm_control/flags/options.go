package flags

import (
	"time"

	logging "github.com/DesistDaydream/logging/pkg/logrus_init"
)

// Flags 是 vm-control 的全部命令行状态。
type Flags struct {
	// 连接
	IPs        []string
	URI        string
	ConfigPath string
	ConfigName string

	// 虚拟机选择
	Domains  []string
	All      bool
	Running  bool
	Stopped  bool
	Parallel bool

	// 关机
	ShutdownWait    bool
	ShutdownForce   bool
	ShutdownTimeout time.Duration

	// 快照
	SnapshotName        string
	SnapshotDescription string
	SnapshotMemory      bool
	SnapshotQuiesce     bool
	SnapshotChildren    bool
	SnapshotForce       bool
	SnapshotRunning     bool
	SnapshotPaused      bool

	// 创建虚拟机
	DomainFile string
	NoStart    bool
	DryRun     bool

	Logging logging.LogrusFlags
}
