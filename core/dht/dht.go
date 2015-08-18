package dht

import (
	"github.com/boreq/netblog/network"
	"github.com/boreq/netblog/network/node"
	"github.com/boreq/netblog/utils"
	"golang.org/x/net/context"
)

var log = utils.GetLogger("dht")

// System-wide replication parameter.
const k = 20

// System-wide concurrency parameter.
const a = 3

func New(ctx context.Context, ident node.Identity, address string) DHT {
	rw := &dht{
		ctx:  ctx,
		net:  network.New(ctx, ident, address),
		rt:   NewRoutingTable(),
		self: ident.Id,
	}
	return rw
}

type dht struct {
	ctx  context.Context
	net  network.Network
	rt   RoutingTable
	self node.ID
}

func (d *dht) Init(nodes []node.NodeInfo) error {
	c, cancel := d.net.Subscribe()
	go func() {
		defer cancel()
		for {
			select {
			case msg := <-c:
				log.Debugf("Received message from %s on %s", msg.Id, msg.Address)
				d.rt.Update(msg.Id, msg.Address)
			case <-d.ctx.Done():
				return
			}
		}
	}()

	err := d.net.Listen()
	if err != nil {
		return err
	}

	for _, nodeInfo := range nodes {
		_, err := d.net.Dial(nodeInfo)
		log.Debugf("Dial %s, err: %s", nodeInfo.Id, err)
	}
	//d.FindNode(d.self)
	return nil
}

func (d *dht) Ping(id node.ID) error {
	return nil
}

func (d *dht) FindNode(id node.ID) (node.NodeInfo, error) {
	//	nodes := d.rt.GetClosest(id, a)

	//	results := make(chan node.NodeInfo)
	// TODO: subscribe

	//	for _, node := range nodes {
	//		peer := d.net.Dial()
	//	}
	return node.NodeInfo{}, nil
}
