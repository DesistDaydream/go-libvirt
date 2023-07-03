package flags

import logging "github.com/DesistDaydream/logging/pkg/logrus_init"

type Flags struct {
	IPs        []string
	ConfigPath string
	ConfigName string
}

type DomainFlags struct {
	DomainsName []string
}

var (
	F  Flags
	DF DomainFlags
	L  logging.LogrusFlags
)
