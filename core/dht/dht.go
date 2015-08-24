package dht

import (
	"container/list"
	"errors"
	"github.com/boreq/netblog/core/dht/kbuckets"
	"github.com/boreq/netblog/network"
	"github.com/boreq/netblog/network/node"
	"github.com/boreq/netblog/protocol/message"
	"github.com/boreq/netblog/utils"
	"golang.org/x/net/context"
	"math/rand"
	"time"
)

// System-wide replication parameter.
const k = 20

// System-wide concurrency parameter.
const a = 3

func New(ctx context.Context, ident node.Identity, address string) DHT {
	rw := &dht{
		ctx:  ctx,
		net:  network.New(ctx, ident, address),
		rt:   kbuckets.New(ident.Id, k),
		self: ident.Id,
	}
	return rw
}

var log = utils.GetLogger("dht")

type dht struct {
	ctx  context.Context
	net  network.Network
	rt   kbuckets.RoutingTable
	self node.ID
}

func (d *dht) Init(nodes []node.NodeInfo) error {
	// Receive all incoming messages to add nodes to the routing table etc.
	go func() {
		c, cancel := d.net.Subscribe()
		defer cancel()
		for {
			select {
			case msg := <-c:
				go d.handleMessage(msg)
			case <-d.ctx.Done():
				return
			}
		}
	}()

	// Listen to incoming connections.
	err := d.net.Listen()
	if err != nil {
		return err
	}

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

	// Init the DHT - run FindNode on local node's id.
	<-time.After(time.Second)
	d.FindNode(d.self)
	return nil
}

func (d *dht) handleMessage(msg network.IncomingMessage) error {
	d.rt.Update(msg.Sender.Id, msg.Sender.Address)

	switch pMsg := msg.Message.(type) {

	case *message.Ping:
		peer, err := d.net.Dial(msg.Sender)
		if err == nil {
			random := pMsg.GetRandom()
			response := &message.Pong{Random: &random}
			peer.Send(response)
		}

	case *message.FindNode:
		nodes := d.rt.GetClosest(pMsg.GetId(), k)
		response := &message.Nodes{}
		for i := 0; i < len(nodes); i++ {
			ndInfo := &message.Nodes_NodeInfo{
				Id:      nodes[i].Id,
				Address: &nodes[i].Address,
			}
			response.Nodes = append(response.Nodes, ndInfo)
		}

		peer, err := d.net.Dial(msg.Sender)
		if err == nil {
			peer.Send(response)
		}
	}
	return nil
}

func (d *dht) Ping(id node.ID) (*time.Duration, error) {
	ctx, cancel := context.WithTimeout(d.ctx, 5*time.Second)
	defer cancel()

	nd, err := d.FindNode(id)
	if err != nil {
		return nil, err
	}

	peer, err := d.net.Dial(nd)
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

	newEntry := NodeData{
		&node.NodeInfo{nd.Id, nd.Address},
		distance,
		false,
	}

	var elem *list.Element = nil
	for elem = l.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(NodeData)
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

func (d *dht) FindNode(id node.ID) (node.NodeInfo, error) {
	log.Debugf("FindNode %s", id)
	ctx, cancel := context.WithTimeout(d.ctx, 20*time.Second)
	defer cancel()

	results := list.New()
	result := make(chan node.NodeInfo)

	// Initial nodes from kbuckets. Take more than 'a' since some of them
	// can be offline.
	nodes := d.rt.GetClosest(id, k)
	if len(nodes) == 0 {
		return node.NodeInfo{}, errors.New("Could not locate the node")
	}
	for _, nd := range nodes {
		if node.CompareId(nd.Id, id) {
			return nd, nil
		}
		AddEntryToList(results, id, &nd)
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
					log.Debug("FindNode response")
					for _, nd := range pMsg.Nodes {
						ndInfo := &node.NodeInfo{nd.GetId(), nd.GetAddress()}
						// If this is the right node just send it to be returned.
						if node.CompareId(nd.GetId(), id) {
							result <- *ndInfo
							return
						}
						// Otherwise add this node to the list of nodes which should be further queried.
						log.Debugf("FindNode new node from response %s", ndInfo.Id)
						AddEntryToList(results, id, ndInfo)
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		// Send new FindNode messages.
		counterSent := 0
		counterIter := 0
		for elem := results.Front(); elem != nil; elem = elem.Next() {
			counterIter++
			if counterSent > a {
				break
			}
			if counterIter > k {
				return node.NodeInfo{}, errors.New("Could not locate the node")
			}
			entry := elem.Value.(NodeData)
			if !entry.Processed {
				entry.Processed = true
				log.Debugf("FindNode dial %s", entry.Info.Id)
				peer, err := d.net.Dial(*entry.Info)
				if err == nil {
					counterSent++
					msg := &message.FindNode{Id: id}
					log.Debugf("FindNode send %x", entry.Info.Id)
					go peer.SendWithContext(ctx, msg)
				}
			}
		}

		// Await results.
		log.Debug("FindNode waiting")
		select {
		case nd := <-result:
			log.Debugf("FindNode result: %s on %s", nd.Id, nd.Address)
			return nd, nil
		case <-ctx.Done():
			return node.NodeInfo{}, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// Calculates the distance between two nodes.
func distance(a, b node.ID) ([]byte, error) {
	return utils.XOR(a, b)
}
