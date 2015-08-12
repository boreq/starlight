package dht

import (
	"github.com/boreq/netblog/network"
	"github.com/boreq/netblog/network/node"
	"github.com/boreq/netblog/utils"
	"golang.org/x/net/context"
)

var log = utils.Logger("dht")

// System-wide replication parameter.
const k = 20
const a = 3

func New(ctx context.Context, ident node.Identity) DHT {
	rw := &dht{
		ctx:  ctx,
		net:  network.New(ctx, ident),
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

func (d *dht) Init(nodes []node.NodeInfo, address string) error {
	log.Printf("Subscribing to messages and running SubGoroutine")
	c, cancel := d.net.Subscribe()
	go func() {
		defer func() {
			log.Print("SubGoroutine close subscription")
			cancel()
		}()
		for {
			select {
			case msg := <-c:
				log.Printf("SubGoroutine received message from %s", msg.Id)
				//d.rt.Update(msg.Id, msg.Address)
			case <-d.ctx.Done():
				log.Print("SubGoroutine context closed")
				return
			}
		}
	}()

	log.Printf("Starting network on %s", address)
	err := d.net.Listen(address)
	if err != nil {
		return err
	}

	log.Print("Initializing")
	for _, nodeInfo := range nodes {
		_, err := d.net.Dial(nodeInfo)
		log.Printf("Dial %s on %s, err: %s", nodeInfo.Id, nodeInfo.Address, err)
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
