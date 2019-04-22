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

// Load loads the specified config file into this struct. Those parameters
// (fields) which are present in that file are overwritten and the rest is
// left untouched.
func (conf *Config) Load(path string) error {
	content, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	return json.Unmarshal(content, conf)
}

// Save saves this struct into the specified config file.
func (conf *Config) Save(path string) error {
	jsonEncoded, err := json.MarshalIndent(conf.savedConfig, "", "	")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, jsonEncoded, 0600)
	return err
}

// Returns already loaded ready-to-use config.
func Get(path string) (*Config, error) {
	conf := Default()
	e := conf.Load(path)
	if e != nil {
		return nil, errors.New("unable to load config")
	}
	return conf, nil
}

// GetDir returns the directory in which config and data should be saved.
func GetDirPath() string {
	// Overriden by env variable
	if envDir := os.Getenv(ConfigEnvVar); envDir != "" {
		return envDir
	}

	// Default directory in $HOME
	user, _ := user.Current()
	return path.Join(user.HomeDir, ".starlight")
}

// GetConfigPath Returns the file in which the config should be saved.
func GetConfigPath() string {
	return path.Join(GetDirPath(), "config.json")
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
