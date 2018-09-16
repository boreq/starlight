package commands

import (
	"github.com/boreq/starlight/config"
	"github.com/boreq/starlight/local"
	"github.com/boreq/starlight/network/node"
	"net/rpc"
)

func GetClient() (*rpc.Client, error) {
	iden, err := node.LoadLocalIdentity(config.GetDir())
	if err != nil {
		return nil, err
	}

	address := local.GetAddress(iden.Id)
	return local.Dial(address)
}
