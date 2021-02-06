package config

import (
	"github.com/jiuzhou-zhao/go-fundamental/clienttoolset"
	"github.com/jiuzhou-zhao/go-fundamental/dbtoolset"
	"github.com/jiuzhou-zhao/go-fundamental/servicetoolset"
)

type Config struct {
	GRpcServerConfig    servicetoolset.GRpcServerConfig
	GRpcClientConfigTpl clienttoolset.GRpcClientConfig
	DBConfig            dbtoolset.DBConfig
}
