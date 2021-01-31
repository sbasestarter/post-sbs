package config

import (
	"github.com/jiuzhou-zhao/go-fundamental/dbtoolset"
	"github.com/jiuzhou-zhao/go-fundamental/servicetoolset"
)

type Config struct {
	GRpcServerConfig servicetoolset.GRpcServerConfig
	DBConfig         dbtoolset.DBConfig

	DiscoveryServerNames map[string]string

	Signer string
}
