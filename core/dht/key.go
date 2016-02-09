package dht

import (
	"errors"
	"github.com/boreq/lainnet/crypto"
	"github.com/boreq/lainnet/network/node"
	"github.com/boreq/lainnet/protocol/message"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
)

func (d *dht) PutPubKey(ctx context.Context, id node.ID, key crypto.PublicKey) error {
	log.Debugf("PutPubKey %s", id)

	// Prepare StorePubKey message.
	keyBytes, err := key.Bytes()
	if err != nil {
		return err
	}
	msg := &message.StorePubKey{Key: keyBytes}

	// Locate the closest nodes.
	nodes, err := d.findNode(ctx, id, false)
	if err != nil {
		return err
	}

	// Send 'k' store RPCs. We don't have to wait for this to finish so
	// a goroutine with the DHT's context is used instead of blocking.
	go func() {
		counter := 0
		for _, nodeInfo := range nodes {
			peer, err := d.netDial(nodeInfo)
			if err == nil {
				err := peer.SendWithContext(d.ctx, msg)
				if err == nil {
					counter++
					if counter > k {
						return
					}
				}
			}
		}
	}()

	return nil
}

func (d *dht) GetPubKey(ctx context.Context, id node.ID) (crypto.PublicKey, error) {
	log.Debugf("GetPubKey %s", id)

	// Try to find the key locally and if it isn't found perform a key
	// lookup procedure.
	key, err := d.getPubKeyLocally(id)
	if err == nil {
		log.Debugf("GetPubKey %s had locally", id)
		return key, nil
	}
	return d.getPubKey(ctx, id)
}

// getPubKeyLocally attempts to return a public key of a node without performing
// a full lookup procedure.
func (d *dht) getPubKeyLocally(id node.ID) (crypto.PublicKey, error) {
	// Check the nodes that we are already communicating with.
	peer, err := d.net.FindActive(id)
	if err == nil {
		return peer.PubKey(), nil
	}

	// Check the public keys datastore.
	key, err := d.pubKeysStore.Get(id)
	if err == nil {
		key, ok := key.(crypto.PublicKey)
		if !ok {
			log.Debug("Error when performing a crypto.PublicKey assertion")
		} else {
			return key, nil
		}
	}

	// Check if it is our key.
	if node.CompareId(id, d.self.Id) {
		return d.self.PubKey, nil
	}

	return nil, errors.New("Public key not found locally")
}

// getPubKey attempts to return a public key of a node by performing a full
// lookup procedure.
func (d *dht) getPubKey(ctx context.Context, id node.ID) (crypto.PublicKey, error) {
	log.Debugf("getPubKey %s", id)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	result := make(chan crypto.PublicKey)

	// Process incoming messages.
	go func() {
		c, cancel := d.net.Subscribe()
		defer cancel()

		for {
			select {
			case msg := <-c:
				switch pMsg := msg.Message.(type) {
				case *message.StorePubKey:
					// TODO inefficient
					pubKey, err := crypto.NewPublicKey(pMsg.GetKey())
					if err == nil {
						keyKey, err := pubKey.Hash()
						if err == nil && node.CompareId(keyKey, id) {
							select {
							case result <- pubKey:
								return
							case <-ctx.Done():
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

	// Run the lookup procedure.
	msgFactory := func(id node.ID) proto.Message {
		rv := &message.FindPubKey{
			Id: id,
		}
		return rv
	}
	go d.findNodeCustom(ctx, id, msgFactory, false)

	// Await results.
	select {
	case key := <-result:
		// Store locally before returning in order to cache the data.
		keyKey, err := key.Hash()
		if err == nil {
			d.pubKeysStore.Store(keyKey, key)
		}
		return key, nil
	case <-ctx.Done():
		return nil, errors.New("Key not found")
	}
}

// handleStorePubKeyMsg processes an incoming StorePubKey message.
func (d *dht) handleStorePubKeyMsg(ctx context.Context, sender node.NodeInfo, msg *message.StorePubKey) error {
	pubKey, err := crypto.NewPublicKey(msg.GetKey())
	if err == nil {
		keyKey, err := pubKey.Hash()
		// A call to node.CompareId below doesn't allow other nodes to
		// republish the data as there is no need to clutter the network
		// with stale data.
		if err == nil && node.CompareId(keyKey, sender.Id) {
			log.Debugf("Storing public key %x", keyKey)
			d.pubKeysStore.Store(keyKey, pubKey)
		}
	}
	return nil
}

// handleFindPubKeyMsg processes an incoming FindPubKey message.
func (d *dht) handleFindPubKeyMsg(ctx context.Context, sender node.NodeInfo, msg *message.FindPubKey) error {
	// Sanity.
	id := msg.GetId()
	if !node.ValidateId(id) {
		return errors.New("Invalid id")
	}

	var response proto.Message = nil
	key, err := d.getPubKeyLocally(id)
	if err == nil {
		log.Debug("FindPubKey response sending the key directly")
		if keyBytes, err := key.Bytes(); err == nil {
			response = &message.StorePubKey{
				Key: keyBytes,
			}
		}
	} else {
		log.Debug("FindPubKey response sending the closest nodes")
		response = d.createNodesMessage(msg.GetId())
	}

	if response != nil {
		peer, err := d.Dial(ctx, sender.Id)
		if err != nil {
			return err
		}
		err = peer.Send(response)
		if err != nil {
			return err
		}
	}
	return nil
}
