package main

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"libvirt.org/go/libvirt"

	humanize "github.com/dustin/go-humanize"
)

// 日志功能初始化
func LogInit(level, file, format string) error {
	// 设置日志格式
	switch format {
	case "text":
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat:   "2006-01-02 15:04:05",
			DisableTimestamp:  false,
			DisableHTMLEscape: false,
			DataKey:           "",
			// FieldMap:          map[logrus.fieldKey]string{},
			// CallerPrettyfier: func(*runtime.Frame) (string, string) {},
			PrettyPrint: false,
		})
	}

	// 设置日志级别
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	logrus.SetLevel(logLevel)

	// 设置日志输出位置，写入到文件或标准输出
	if file != "" {
		f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			return err
		}
		logrus.SetOutput(f)
	}

	return nil
}

// 获取所有状态的虚拟机
func GetAllDoms(conn *libvirt.Connect) ([]libvirt.Domain, error) {
	activeDoms, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE)
	if err != nil {
		return nil, err
	}
	inactiveDoms, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
	if err != nil {
		return nil, err
	}

	logrus.Infof("当前有 %d 台正在运行的虚拟机", len(activeDoms))
	logrus.Infof("当前有 %d 台正在关闭的虚拟机", len(inactiveDoms))

	doms := append(activeDoms, inactiveDoms...)
	return doms, nil
}

// 获取待操作的虚拟机
func GetDoms(conn *libvirt.Connect, names []string) ([]libvirt.Domain, error) {
	var doms []libvirt.Domain

	for _, name := range names {
		dom, _ := conn.LookupDomainByName(name)
		doms = append(doms, *dom)
	}

	return doms, nil
}

// 获取虚拟机简要信息
func GetInfo(conn *libvirt.Connect) {
	doms, err := GetAllDoms(conn)
	if err != nil {
		logrus.Panic("获取虚拟机信息异常: ", err)
	}

	for _, dom := range doms {
		d, err := dom.GetInfo()
		if err != nil {
			logrus.Error("获取虚拟机信息异常: ", err)
		}

		name, err := dom.GetName()
		if err != nil {
			logrus.Error("获取虚拟机名称异常: ", err)
		}

		logrus.WithFields(logrus.Fields{
			"状态":      d.State,
			"可调整最大内存": humanize.IBytes(d.MaxMem * 1024),
			"已配置内存":   humanize.IBytes(d.Memory * 1024),
			"逻辑CPU数":  d.NrVirtCpu,
			"CPU运行时间": time.Nanosecond * time.Duration(d.CpuTime),
		}).Infof("%v 虚拟机信息", name)

		dom.Free()
	}
}

// 关闭虚拟机
func Close(doms []libvirt.Domain) error {
	for _, dom := range doms {
		name, err := dom.GetName()
		if err != nil {
			logrus.Error("获取虚拟机名称异常: ", err)
			return err
		}

		err = dom.Shutdown()
		if err != nil {
			logrus.Error("关闭虚拟机异常: ", err)
			return err
		}

		logrus.Infof("%v 虚拟机已关闭", name)
		dom.Free()
	}

	return nil
}

// 启动虚拟机
func Start(doms []libvirt.Domain) error {
	for _, dom := range doms {
		name, err := dom.GetName()
		if err != nil {
			logrus.Error("获取虚拟机名称异常: ", err)
			return err
		}

		err = dom.Create()
		if err != nil {
			logrus.Error("启动虚拟机异常: ", err)
			return err
		}

		logrus.Infof("%v 虚拟机已启动", name)
		dom.Free()
	}

	return nil
}

// 创建快照
func CreateSnapshot(doms []libvirt.Domain) error {
	return nil
}

func main() {
	logLevel := pflag.StringP("log-level", "l", "info", "日志级别:[debug, info, warn, error, fatal]")
	logFile := pflag.StringP("log-output", "o", "", "日志输出文件位置,默认为空,输出到标准输出")
	logFormat := pflag.StringP("log-format", "f", "text", "日志输出格式: text, json")

	target := pflag.StringP("target", "t", "qemu:///system", "要连接的QEMU服务器")
	operation := pflag.StringP("operation", "p", "", "对虚拟机执行的操作: start, close, info")

	pflag.Parse()

	if err := LogInit(*logLevel, *logFile, *logFormat); err != nil {
		logrus.Fatal("set log level error")
	}

	// 连接到 libvirtd
	conn, err := libvirt.NewConnect(*target)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	if operation == nil || *operation == "" {
		logrus.Fatal("请指定操作")
	}

	switch *operation {
	case "start":
	case "close":
	case "info":
		GetInfo(conn)
	default:
		logrus.Fatal("不支持的操作")
	}

}
