package dht

import (
	"container/list"
	"crypto"
	"errors"
	"github.com/boreq/lainnet/core/dht/channelstore"
	"github.com/boreq/lainnet/core/dht/datastore"
	"github.com/boreq/lainnet/core/dht/kbuckets"
	"github.com/boreq/lainnet/network"
	"github.com/boreq/lainnet/network/dispatcher"
	"github.com/boreq/lainnet/network/node"
	"github.com/boreq/lainnet/protocol/message"
	"github.com/boreq/lainnet/utils"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"math/rand"
	"time"
)

// System-wide replication parameter.
const k = 20

// System-wide concurrency parameter.
const a = 3

// Hash used for signing messages stored in the DHT (for example StoreChannel
// message).
const signingHash = crypto.SHA256

// Stored public keys will be removed after this time passes.
var pubKeyStoreTimeout = 2 * time.Hour

// Stored channel memberships will be removed/rejected after this time passes
// since they have been signed.
var channelStoreTimeout = 5 * time.Minute

var log = utils.GetLogger("dht")

func New(ctx context.Context, net network.Network, ident node.Identity) DHT {
	rw := &dht{
		ctx:          ctx,
		net:          net,
		rt:           kbuckets.New(ident.Id, k),
		self:         ident,
		disp:         dispatcher.New(ctx),
		pubKeysStore: datastore.New(pubKeyStoreTimeout),
		channelStore: channelstore.New(channelStoreTimeout),
	}
	return rw
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

func (d *dht) Init(nodes []node.NodeInfo) error {
	// Receive all incoming messages to add nodes to the routing table etc.
	go func() {
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
	}()

	// Init the DHT - populate buckets with initial nodes.
	for _, nodeInfo := range nodes {
		peer, err := d.net.Dial(nodeInfo)
		if err != nil {
			log.Debugf("Init dial %s, err: %s", nodeInfo.Id, err)
			continue
		}
		random := rand.Uint32()
		msg := &message.Ping{Random: &random}
		err = peer.Send(msg)
		if err != nil {
			log.Debugf("Init ping %s, err: %s", nodeInfo.Id, err)
			continue
		}
	}

	// Init the DHT - refresh the more distant buckets.
	// TODO

	// Init the DHT - run FindNode on local node's id.
	<-time.After(time.Second)
	d.findNode(d.ctx, d.self.Id, false)

	// Init the DHT - run the bootstrap once before returning and then
	// continue in a loop.
	err := d.bootstrap(d.ctx)
	if err != nil {
		return err // TODO don't abort I think
	}
	go d.runBootstrap(d.ctx, time.Hour)

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
	// TODO

	// Republish.
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
	return d.net.Dial(nd)
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

type NodeData struct {
	Info      *node.NodeInfo
	Distance  []byte
	Processed bool
}

func AddEntryToList(l *list.List, id node.ID, nd *node.NodeInfo) {
	distance, err := distance(id, nd.Id)
	if err != nil {
		return
	}

	newEntry := &NodeData{
		&node.NodeInfo{nd.Id, nd.Address},
		distance,
		false,
	}

	var elem *list.Element = nil
	for elem = l.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(*NodeData)
		// If new one is closer insert it before this element.
		res, _ := utils.Compare(distance, entry.Distance)
		if res > 0 {
			break
		}
	}

	if elem != nil {
		l.InsertBefore(newEntry, elem)
	} else {
		l.PushBack(newEntry)
	}
}

func (d *dht) FindNode(ctx context.Context, id node.ID) (node.NodeInfo, error) {
	log.Debugf("FindNode %s", id)
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	nodes, err := d.findNode(ctx, id, true)
	if err != nil {
		return node.NodeInfo{}, err
	}
	log.Debugf("FindNode got %d results", len(nodes))
	if len(nodes) > 0 && node.CompareId(nodes[0].Id, id) {
		return nodes[0], nil
	} else {
		return node.NodeInfo{}, errors.New("Node not found")
	}
}

type messageFactory func(id node.ID) proto.Message

// createNodesMessage creates a Nodes message with the 'k' known nodes closest
// to the provided id.
func (d *dht) createNodesMessage(id node.ID) *message.Nodes {
	nodes := d.rt.GetClosest(id, k)
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

func (d *dht) findNode(ctx context.Context, id node.ID, breakOnResult bool) ([]node.NodeInfo, error) {
	msgFactory := func(id node.ID) proto.Message {
		rv := &message.FindNode{
			Id: id,
		}
		return rv
	}
	return d.findNodeCustom(ctx, id, msgFactory, breakOnResult)
}

// findNode attempts to locate k closest nodes to a given key. If breakOnResult
// is true the lookup will stop immidiately when a result with a matching key
// is found. Otherwise the lookup will continue anyway to find k closest nodes.
func (d *dht) findNodeCustom(ctx context.Context, id node.ID, msgFac messageFactory, breakOnResult bool) ([]node.NodeInfo, error) {
	log.Debugf("findNode %s, break: %t", id, breakOnResult)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := list.New()
	result := make(chan *list.List)

	// Initial nodes from kbuckets. Take more than 'a' since some of them
	// can be offline.
	nodes := d.rt.GetClosest(id, k)
	if len(nodes) == 0 {
		log.Debug("findNode buckets empty, aborting")
		return nodeDataListToSlice(results), errors.New("Could not locate the node")
	}
	for _, nd := range nodes {
		AddEntryToList(results, id, &nd)
	}
	if elem := results.Front(); elem != nil && node.CompareId(elem.Value.(*NodeData).Info.Id, id) {
		log.Debug("findNode found in buckets")
		return nodeDataListToSlice(results), nil
	}

	// Handle incoming messages.
	go func() {
		c, cancel := d.net.Subscribe()
		defer cancel()
		for {
			select {
			case msg := <-c:
				switch pMsg := msg.Message.(type) {
				case *message.Nodes:
					for _, nd := range pMsg.Nodes {
						// Add to the list.
						ndInfo := &node.NodeInfo{nd.GetId(), nd.GetAddress()}
						if !node.CompareId(ndInfo.Id, d.self.Id) {
							AddEntryToList(results, id, ndInfo)
							log.Debugf("findNode new node from response %s", ndInfo.Id)
							// If this is the right node return the results already.
							// TODO fix this
							if breakOnResult && node.CompareId(nd.GetId(), id) {
								result <- results
								return
							}
						}
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	var noBetterResults bool = false

	for {
		// Send new FindNode messages.
		counterSent := 0
		counterIter := 0
		for elem := results.Front(); elem != nil; elem = elem.Next() {
			counterIter++
			// Stop after sending 'a' messages and await results.
			if counterSent > a {
				break
			}

			// Stop if messages were already sent to 'k' closest nodes.
			if counterIter > k {
				return nodeDataListToSlice(results), nil
			}

			// Iterated over all results and nothing was sent more
			// than on time in a row - nothing new was received.
			if counterSent == 0 && elem.Next() == nil {
				if !noBetterResults {
					noBetterResults = true
				} else {
					return nodeDataListToSlice(results), nil
				}
			}

			// Send new messages.
			entry := elem.Value.(*NodeData)
			if !entry.Processed {
				entry.Processed = true
				log.Debugf("findNode dial %s", entry.Info.Id)
				peer, err := d.net.Dial(*entry.Info)
				if err == nil {
					counterSent++
					msg := &message.FindNode{Id: id}
					log.Debugf("findNode send to %s", entry.Info.Id)
					go peer.SendWithContext(ctx, msg)
				}
			}
		}

		// Await results.
		log.Debug("findNode waiting")
		select {
		case <-result:
			return nodeDataListToSlice(results), nil
		case <-ctx.Done():
			return nodeDataListToSlice(results), ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}

func nodeDataListToSlice(l *list.List) []node.NodeInfo {
	rv := make([]node.NodeInfo, l.Len())
	i := 0
	for elem := l.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(*NodeData)
		rv[i] = *entry.Info
		i++
	}
	return rv
}

// Calculates the distance between two nodes.
func distance(a, b node.ID) ([]byte, error) {
	return utils.XOR(a, b)
}
