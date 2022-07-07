package config

import (
	"github.com/sgostarter/libservicetoolset/clienttoolset"
	"github.com/sgostarter/libservicetoolset/servicetoolset"
)

type Config struct {
	GRpcServerConfig    servicetoolset.GRPCServerConfig `yaml:"grpc_server_config"`
	GRpcClientConfigTpl clienttoolset.GRPCClientConfig  `yaml:"grpc_client_config_tpl"`
	RedisDSN            string                          `yaml:"redis_dsn"`

	DiscoveryServerNames map[string]string `yaml:"discovery_server_names"`

	Signer string `yaml:"signer"`
}
