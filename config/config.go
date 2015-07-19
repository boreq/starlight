package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"os/user"
	"path"
)

// This part of the config structure is saved in the config file in JSON format.
type savedConfig struct {
	ListenAddress string
	LocalAddress  string
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
	if envDir := os.Getenv("NETBLOGPATH"); envDir != "" {
		return envDir
	}

	// Default directory in $HOME
	user, _ := user.Current()
	return path.Join(user.HomeDir, ".netblog")
}

// Returns a config filled with default values.
func Default() *Config {
	conf := &Config{
		savedConfig{
			ListenAddress: ":1836",
			LocalAddress:  "/tmp/netblog.socket",
		},
	}
	return conf
}
