package commands

import (
	"github.com/boreq/starlight/config"
	"github.com/boreq/starlight/local"
	"github.com/boreq/starlight/network/node"
	"net/rpc"
)

func GetClient() (*rpc.Client, error) {
	iden, err := GetIdentity()
	if err != nil {
		return nil, err
	}

	address := local.GetAddress(iden.Id)
	return local.Dial(address)
}

func GetConfig() (*config.Config, error) {
	path := config.GetConfigPath()
	return config.Get(path)
}

func GetIdentity() (*node.Identity, error) {
	path := config.GetDirPath()
	return node.LoadLocalIdentity(path)
}
