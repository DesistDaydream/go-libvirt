package flags

import logging "github.com/DesistDaydream/logging/pkg/logrus_init"

type Flags struct {
	IPs        []string
	Test       string
	ConfigPath string
	ConfigName string
}

type DomainFlags struct {
	DomainsName []string
}

var (
	F  Flags
	DF DomainFlags
	LF logging.LogrusFlags
)
