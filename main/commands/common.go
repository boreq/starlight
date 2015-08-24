package commands

import (
	"github.com/boreq/netblog/config"
	"github.com/boreq/netblog/local"
	"github.com/boreq/netblog/network/node"
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
