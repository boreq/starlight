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
	LocalAddress string
}

// Full config struct.
type Config struct {
	savedConfig
	// Directory containing all configuration and data files.
	Directory string
}

func (conf *Config) filePath() string {
	return path.Join(conf.Directory, "config.json")
}

func (conf *Config) Load() error {
	content, e := ioutil.ReadFile(conf.filePath())
	if os.IsNotExist(e) {
		return nil
	}
	return json.Unmarshal(content, conf)
}

func (conf *Config) Save() error {
	err := os.MkdirAll(conf.Directory, 0700)
	if err != nil && !os.IsExist(err) {
		return err
	}
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

// Returns a config filled with default values.
func Default() *Config {
	// Default directory in $HOME
	user, _ := user.Current()
	dir := path.Join(user.HomeDir, ".netblog")

	// Overriden by env variable
	envDir := os.Getenv("NETBLOGPATH")
	if envDir != "" {
		dir = envDir
	}

	conf := &Config{
		savedConfig{
			ListenAddress: ":1836",
			LocalAddress: "/tmp/netblog.socket",
		},
		dir,
	}
	return conf
}
