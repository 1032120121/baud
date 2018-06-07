package router

import (
	"github.com/BurntSushi/toml"
	"github.com/tiglabs/baudengine/util/log"
	"time"
)

const defaultConfig = `
# Router Configuration.

[module]
role = "router"
clusterId = "1"
ip = "0.0.0.0"
httpPort = 9000
pprof = 10088
masterAddr = "localhost:18817"
logDir = "/export/log/ps"
masterConnPoolSize = 10
psConnPoolSize = 10

[log]
log-path = "/tmp/baudengine/router/log"
#debug, info, warn, error
level="debug"
`

const rpcTimeoutDef  = 5 * time.Second

type ModuleConfig struct {
	Role               string `toml:"role,omitempty" json:"role"`
	ClusterId          string `toml:"clusterId,omitempty" json:"clusterId"`
	Ip                 string
	HttpPort           uint32
	Pprof              uint32
	MasterAddr         string
	MasterConnPoolSize uint16
	PsConnPoolSize     uint16
}

type LogConfig struct {
	LogPath   string `toml:"log-path,omitempty" json:"log-path"`
	Level     string `toml:"level,omitempty" json:"level"`
}

type Config struct {
	ModuleCfg  ModuleConfig  `toml:"module,omitempty" json:"module"`
	LogCfg     LogConfig     `toml:"log,omitempty" json:"log"`
}

func LoadConfig(fileName string) *Config {
	config := &Config{}
	if _, err := toml.Decode(defaultConfig, config); err != nil {
		log.Panic("decode defaultConfig failed, err %v", err)
	}
	if fileName != "" {
		if err := config.LoadFromFile(fileName); err != nil {
			log.Panic("decode %s failed, err %v", fileName, err)
		}
	}
	return config
}

func (config *Config) LoadFromFile(fileName string) error {
	if _, err := toml.DecodeFile(fileName, config); err != nil {
		return err
	}
	return config.validate()
}

func (config *Config) validate() error {
	return nil
}
