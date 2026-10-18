package app

import (
	"errors"
	"fmt"
	"io"
	"os"

	logging "github.com/DesistDaydream/logging/pkg/logrus_init"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/DesistDaydream/go-libvirt/cmd/vm_control/flags"
	"github.com/DesistDaydream/go-libvirt/pkg/conf"
	"github.com/DesistDaydream/go-libvirt/pkg/report"
	"github.com/DesistDaydream/go-libvirt/pkg/virt"
)

// App 保存一次命令执行需要的全部运行时状态。
type App struct {
	Flags *flags.Flags
	Pool  *virt.Pool
	Out   io.Writer
}

func New() *App {
	return &App{
		Flags: &flags.Flags{},
		Out:   os.Stdout,
	}
}

// AddRootFlags 注册所有子命令共享的标志。
func (a *App) AddRootFlags(cmd *cobra.Command) {
	pf := cmd.PersistentFlags()
	pf.StringSliceVar(&a.Flags.IPs, "ips", nil, "宿主机地址列表，逗号分隔；URI 模板中的 {host} 会逐个替换")
	pf.StringVar(&a.Flags.URI, "uri", "", "libvirt 连接 URI 模板（qemu+tcp/qemu+ssh 等），默认 "+virt.DefaultURI)
	pf.StringVar(&a.Flags.ConfigPath, "config-path", ".", "配置文件路径")
	pf.StringVar(&a.Flags.ConfigName, "config-name", "my_config", "配置文件名称，不含扩展名")
	a.addLoggingFlags(pf)
}

func (a *App) addLoggingFlags(pf *pflag.FlagSet) {
	lf := &a.Flags.Logging
	pf.StringVar(&lf.LogLevel, "log-level", "info", "日志级别 [trace, debug, info, warn, error, fatal, panic]")
	pf.StringVar(&lf.LogOutput, "log-output", "", "日志输出文件，留空输出到标准错误")
	pf.StringVar(&lf.LogFormat, "log-format", "text", "日志格式 [text, json]")
	pf.BoolVar(&lf.LogCaller, "log-caller", false, "输出调用者信息")
	pf.BoolVar(&lf.LogColors, "log-colors", false, "关闭颜色输出")
	pf.IntVar(&lf.LogRotateMaxSize, "log-rotate-max-size", 5, "日志文件最大 MiB")
	pf.IntVar(&lf.LogRotateBackups, "log-rotate-backups", 5, "日志文件最大备份数")
	pf.IntVar(&lf.LogRotateMaxAge, "log-rotate-max-age", 1, "日志文件最大保留天数")
	pf.BoolVar(&lf.LogRotateCompress, "log-rotate-compress", true, "压缩归档日志")
}

// Init 由 domain 命令的 PersistentPreRunE 调用：初始化日志、读取配置并连接宿主机。
func (a *App) Init(_ *cobra.Command, _ []string) error {
	if err := logging.LogrusInit(&a.Flags.Logging); err != nil {
		return fmt.Errorf("初始化日志失败: %w", err)
	}

	if err := a.readConfig(); err != nil {
		return err
	}

	pool, err := virt.Dial(a.Flags.IPs, a.Flags.URI)
	if err != nil {
		return err
	}
	a.Pool = pool
	return nil
}

func (a *App) readConfig() error {
	cfg, err := conf.LoadMyConfig(a.Flags.ConfigPath, a.Flags.ConfigName)
	if err != nil {
		return err
	}

	if len(a.Flags.IPs) == 0 {
		a.Flags.IPs = cfg.IPs
	}
	if a.Flags.URI == "" {
		a.Flags.URI = cfg.URI
	}
	if a.Flags.URI == "" {
		a.Flags.URI = virt.DefaultURI
	}
	if len(a.Flags.Domains) == 0 {
		a.Flags.Domains = cfg.Domains
	}

	return nil
}

func (a *App) Close() {
	if a.Pool != nil {
		a.Pool.Close()
		a.Pool = nil
	}
}

// HostFor 按 domain list 中的 host 选择连接；未指定时要求只有一台宿主机。
func (a *App) HostFor(name string) (*virt.Host, error) {
	hosts := a.Pool.Hosts()
	if len(hosts) == 0 {
		return nil, errors.New("没有可用的宿主机连接")
	}
	if name == "" {
		if len(hosts) == 1 {
			return hosts[0], nil
		}
		return nil, errors.New("domain list 未指定 host，且当前连接了多台宿主机")
	}
	for _, host := range hosts {
		if host.Addr == name || host.Name == name {
			return host, nil
		}
	}
	return nil, fmt.Errorf("未找到宿主机 %q", name)
}

// Select 校验选择参数并返回选中的虚拟机。
func (a *App) Select(require bool) (*virt.Selection, error) {
	if err := a.validateSelection(require); err != nil {
		return nil, err
	}
	return virt.Select(a.Pool.Hosts(), a.SelectOptions())
}

func (a *App) SelectOptions() virt.SelectOptions {
	return virt.SelectOptions{
		Names:   a.Flags.Domains,
		All:     a.Flags.All,
		Running: a.Flags.Running,
		Stopped: a.Flags.Stopped,
	}
}

func (a *App) validateSelection(require bool) error {
	if a.Flags.All && len(a.Flags.Domains) > 0 {
		return errors.New("--all 与 --domains 不能同时使用")
	}
	if a.Flags.Running && a.Flags.Stopped {
		return errors.New("--running 与 --stopped 不能同时使用")
	}
	if require && !a.Flags.All && len(a.Flags.Domains) == 0 {
		return errors.New("请用 --domains 指定虚拟机，或用 --all 选择全部")
	}
	return nil
}

// RunBatch 执行批量操作、输出结果表，并把失败汇总成非零退出码。
func (a *App) RunBatch(action string, selection *virt.Selection, fn virt.Action) error {
	results := virt.Run(selection.Domains, action, fn, a.Flags.Parallel)
	for _, pattern := range selection.Unmatched {
		results = append(results, virt.Result{
			Domain: pattern,
			Action: action,
			Err:    errors.New("未找到匹配的虚拟机"),
		})
	}

	if len(results) == 0 {
		return errors.New("没有匹配的虚拟机")
	}
	failures, err := report.Results(a.Out, results)
	if err != nil {
		return err
	}
	if failures > 0 {
		return fmt.Errorf("%d 个虚拟机操作失败", failures)
	}
	return nil
}
