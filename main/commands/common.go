package commands

import (
	"github.com/boreq/lainnet/config"
	"github.com/boreq/lainnet/local"
	"github.com/boreq/lainnet/network/node"
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
