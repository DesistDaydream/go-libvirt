package flags

import logging "github.com/DesistDaydream/logging/pkg/logrus_init"

type Flags struct {
	Hosts      []string
	ConfigPath string
	ConfigName string
}

var (
	F Flags
	L logging.LogrusFlags
)
