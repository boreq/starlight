package config

import (
	"encoding/json"
	"errors"
	"github.com/boreq/starlight/network/node"
	"io/ioutil"
	"os"
	"os/user"
	"path"
)

// The name of the environment variable which specifies the location of the
// config directory.
const ConfigEnvVar = "STARLIGHTPATH"

// This part of the config structure is saved in the config file in JSON format.
type savedConfig struct {
	ListenAddress     string
	IRCGatewayAddress string
	BootstrapNodes    []node.NodeInfo
}

// Full config struct.
type Config struct {
	savedConfig
}

func (conf *Config) filePath() string {
	return path.Join(GetDir(), "config.json")
}

func (conf *Config) Load() error {
	content, e := ioutil.ReadFile(conf.filePath())
	if os.IsNotExist(e) {
		return nil
	}
	return json.Unmarshal(content, conf)
}

func (conf *Config) Save() error {
	jsonEncoded, err := json.MarshalIndent(conf.savedConfig, "", "	")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(conf.filePath(), jsonEncoded, 0600)
	return err
}

// Returns already loaded ready-to-use config.
func Get() (*Config, error) {
	conf := Default()
	e := conf.Load()
	if e != nil {
		return nil, errors.New("Unable to load config")
	}
	return conf, nil
}

// Returns the directory in which config and data should be saved.
func GetDir() string {
	// Overriden by env variable
	if envDir := os.Getenv(ConfigEnvVar); envDir != "" {
		return envDir
	}

	// Default directory in $HOME
	user, _ := user.Current()
	return path.Join(user.HomeDir, ".starlight")
}

func getDefaultBootstrap() []node.NodeInfo {
	def := map[string]string{
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855": "address:1836",
	}

	var rw []node.NodeInfo
	for id, address := range def {
		nodeId, err := node.NewId(id)
		if err == nil {
			rw = append(rw, node.NodeInfo{Id: nodeId, Address: address})
		}
	}
	return rw
}

// Returns a config filled with default values.
func Default() *Config {
	conf := &Config{
		savedConfig{
			ListenAddress:     ":1836",
			IRCGatewayAddress: ":6667",
			BootstrapNodes:    getDefaultBootstrap(),
		},
	}
	return conf
}
