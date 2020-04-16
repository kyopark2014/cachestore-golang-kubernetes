package config

import (
	"cachestore-golang-kubernetes/internal/log"
	"encoding/json"
	"io/ioutil"
)

// Config is the only one instance holding configuration
// of this service.
var config *AppConfig

// SQLConfig defines the parameters for SQL DB
type SQLConfig struct {
	Host     string `json:"Host"`
	Port     string `json:"Port"`
	Username string `json:"Username"`
	Password string `json:"Password"`
	Database string `json:"Database"`
	Protocol string `json:"Protocol"`
}

// AppConfig is a structure into which config file
// (e.g., config/config.json) is loaded.
type AppConfig struct {
	Logging struct {
		Enable bool   `json:"Enable"`
		Level  string `json:"Level"`
	} `json:"Logging"`

	GracefulTermTimeMillis int64
	Redis                  RedisConfig
	SQL                    SQLConfig
}

// RedisConfig is for parameters of Redis
type RedisConfig struct {
	Host            string
	ReaderHost      string
	Port            string
	PoolMaxIdle     int
	PoolMaxActive   int
	PoolIdleTimeout int
	TTL             int
	Password        string
	ConnTimeout     int
}

// GetInstance returns the pointer to the singleton instance of Config
func GetInstance() *AppConfig {
	if config == nil {
		config = &AppConfig{}
	}
	return config
}

// Load reads config file (e.g., configs/config.json) and
// unmarshalls JSON string in it into Config structure
func (AppConfig) Load(fname string) bool {
	log.D("Load config from the file \"" + fname + "\".")

	b, err := ioutil.ReadFile(fname)
	if err != nil {
		log.E("%v", err)
		return false
	}

	errCode := json.Unmarshal(b, &config)
	log.D("config: %v , err: %v", config, errCode)

	return true
}
