package dht

import (
	"crypto"
	"github.com/boreq/starlight/core/dht/channelstore"
	"github.com/boreq/starlight/core/dht/datastore"
	"github.com/boreq/starlight/core/dht/kbuckets"
	"github.com/boreq/starlight/network"
	"github.com/boreq/starlight/network/dispatcher"
	"github.com/boreq/starlight/network/node"
	"github.com/boreq/starlight/protocol/message"
	"github.com/boreq/starlight/utils"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"math/rand"
	"time"
)

// System-wide replication parameter.
const paramK = 20

// System-wide concurrency parameter.
const paramA = 3

// The lookup procedure is performed over 'd' disjoint paths. As 'k' closest
// nodes are split between 'd' buckets and queried 'a' nodes at a time it is
// good to have those numbers have meaningful values in relation to each
// other.
const paramD = 2

// Hash used for signing messages stored in the DHT (for example StoreChannel
// messages).
const SigningHash = crypto.SHA256

// Stored public keys will be removed after this time passes.
const pubKeyStoreTimeout = 2 * time.Hour

// How often the bootstrap procedure should run.
const bootstrapInterval = 1 * time.Hour

// How often should a bucket be refreshed if no lookup procedure was performed
// on the nodes falling within its range.
const refreshbucketsAfter = 1 * time.Hour

var log = utils.GetLogger("dht")

func New(ctx context.Context, net network.Network, ident node.Identity) DHT {
	rv := &dht{
		ctx:          ctx,
		net:          net,
		rt:           kbuckets.New(ident.Id, paramK, refreshbucketsAfter),
		self:         ident,
		disp:         dispatcher.New(ctx),
		pubKeysStore: datastore.New(pubKeyStoreTimeout),
		channelStore: channelstore.New(maxStoreChannelMessageAge),
	}
	go rv.listenToNetwork()
	return rv
}

type dht struct {
	ctx          context.Context
	net          network.Network
	rt           kbuckets.RoutingTable
	self         node.Identity
	disp         dispatcher.Dispatcher
	pubKeysStore *datastore.Datastore
	channelStore *channelstore.Channelstore
}

func (d *dht) Subscribe() (chan dispatcher.IncomingMessage, dispatcher.CancelFunc) {
	return d.disp.Subscribe()
}

func (d *dht) listenToNetwork() {
	c, cancel := d.net.Subscribe()
	defer cancel()
	for {
		select {
		case msg := <-c:
			go d.handleMessage(d.ctx, msg)
		case <-d.ctx.Done():
			return
		}
	}
}

func (d *dht) Init(nodes []node.NodeInfo) error {
	// Init the DHT - insert all defined bootstrap nodes into the buckets.
	for _, nodeInfo := range nodes {
		d.rt.Update(nodeInfo.Id, nodeInfo.Address)
	}

	// Init the DHT - refresh the more distant buckets.
	// TODO

	// Init the DHT - run FindNode on local node's id.
	_, err := d.findNode(d.ctx, d.self.Id, false)
	if err != nil {
		return errors.Wrap(err, "findNode on local id failed")
	}

	// Init the DHT - run the bootstrap once before returning and then
	// continue in a loop.
	err = d.bootstrap(d.ctx)
	if err != nil {
		return errors.Wrap(err, "initial bootstrap failed")
	}
	go d.runBootstrap(d.ctx, bootstrapInterval)

	return nil
}

func (d *dht) runBootstrap(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			d.bootstrap(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (d *dht) bootstrap(ctx context.Context) error {
	log.Debug("bootstrap")

	// Refresh buckets.
	ids := d.rt.GetForRefresh()
	for _, id := range ids {
		log.Debugf("bootstrap findNode %s", id)
		go d.findNode(ctx, id, false)
	}

	// Republish local node's public key.
	err := d.PutPubKey(ctx, d.self.Id, d.self.PubKey)
	if err != nil {
		return err
	}

	return nil
}

func (d *dht) handleMessage(ctx context.Context, msg dispatcher.IncomingMessage) error {
	d.rt.Update(msg.Sender.Id, msg.Sender.Address)

	switch pMsg := msg.Message.(type) {

	case *message.Ping:
		peer, err := d.Dial(ctx, msg.Sender.Id)
		if err == nil {
			random := pMsg.GetRandom()
			response := &message.Pong{Random: &random}
			peer.Send(response)
		}

	case *message.FindNode:
		response := d.createNodesMessage(pMsg.GetId())
		peer, err := d.Dial(ctx, msg.Sender.Id)
		if err == nil {
			peer.Send(response)
		}

	case *message.StorePubKey:
		d.handleStorePubKeyMsg(ctx, msg.Sender, pMsg)

	case *message.FindPubKey:
		d.handleFindPubKeyMsg(ctx, msg.Sender, pMsg)

	case *message.StoreChannel:
		d.handleStoreChannelMsg(ctx, msg.Sender, pMsg)

	case *message.FindChannel:
		d.handleFindChannelMsg(ctx, msg.Sender, pMsg)

	case *message.PrivateMessage:
		go d.disp.Dispatch(msg.Sender, pMsg)

	case *message.ChannelMessage:
		go d.disp.Dispatch(msg.Sender, pMsg)

	}
	return nil
}

func (d *dht) Dial(ctx context.Context, id node.ID) (network.Peer, error) {
	p, err := d.net.FindActive(id)
	if err == nil {
		return p, nil
	}
	nd, err := d.FindNode(ctx, id)
	if err != nil {
		return nil, err
	}
	return d.netDial(nd)
}

// netDial wraps net.Dial in order to remove a node from the buckets if it fails
// to respond or returns a different error.
func (d *dht) netDial(nd node.NodeInfo) (network.Peer, error) {
	p, err := d.net.Dial(nd)
	if err != nil {
		d.rt.Unresponsive(nd.Id, nd.Address)
	}
	return p, err
}

// netCheckOnline wraps net.CheckOnline in order to remove a node from the
// buckets if it fails to respond or returns a different error.
func (d *dht) netCheckOnline(ctx context.Context, nd node.NodeInfo) error {
	err := d.net.CheckOnline(ctx, nd)
	if err != nil {
		d.rt.Unresponsive(nd.Id, nd.Address)
	}
	return err
}

func (d *dht) Ping(ctx context.Context, id node.ID) (*time.Duration, error) {
	ctx, cancel := context.WithTimeout(d.ctx, 5*time.Second)
	defer cancel()

	peer, err := d.Dial(ctx, id)
	if err != nil {
		return nil, err
	}

	result := make(chan *message.Pong)

	random := rand.Uint32()
	go func() {
		defer close(result)
		c, cancel := d.net.Subscribe()
		defer cancel()

		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-c:
				if node.CompareId(msg.Sender.Id, id) {
					pMsg, ok := msg.Message.(*message.Pong)
					if ok && pMsg.GetRandom() == random {
						select {
						case <-ctx.Done():
							return
						case result <- pMsg:
						}
					}
				}
			}
		}
	}()

	msg := &message.Ping{Random: &random}
	err = peer.SendWithContext(ctx, msg)
	if err != nil {
		return nil, err
	}
	start := time.Now()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-result:
		duration := time.Since(start)
		return &duration, nil
	}
}

// createNodesMessage creates a Nodes message with the 'k' known nodes closest
// to the provided id.
func (d *dht) createNodesMessage(id node.ID) *message.Nodes {
	nodes := d.rt.GetClosest(id, paramK)
	msg := &message.Nodes{}
	for i := 0; i < len(nodes); i++ {
		ndInfo := &message.Nodes_NodeInfo{
			Id:      nodes[i].Id,
			Address: &nodes[i].Address,
		}
		msg.Nodes = append(msg.Nodes, ndInfo)
	}
	return msg
}
