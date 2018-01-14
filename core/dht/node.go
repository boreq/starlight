package dht

import (
	"container/list"
	"errors"
	"github.com/boreq/starlight/network/node"
	"github.com/boreq/starlight/protocol/message"
	"github.com/boreq/starlight/utils"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"sync"
	"time"
)

func (d *dht) FindNode(ctx context.Context, id node.ID) (node.NodeInfo, error) {
	log.Debugf("FindNode %s", id)
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	// Run the lookup procedure.
	nodes, err := d.findNode(ctx, id, true)
	if err != nil {
		return node.NodeInfo{}, err
	}
	log.Debugf("FindNode got %d results", len(nodes))

	// Check if the procedure managed to locate the right node.
	if len(nodes) > 0 && node.CompareId(nodes[0].Id, id) {
		return nodes[0], nil
	} else {
		return node.NodeInfo{}, errors.New("Node not found")
	}
}

// messageFactory is used to send different messages during the node lookup
// procedure - for example FindPubKey is used instead of FindNode during a key
// lookup, so the calling function has to provide a way of generating that
// message instead of the default one. The provided node ID is the id of the
// searched node.
type messageFactory func(id node.ID) proto.Message

// findNode performs a standard node lookup procedure using the FindNode
// message.
func (d *dht) findNode(ctx context.Context, id node.ID, breakOnResult bool) ([]node.NodeInfo, error) {
	msgFactory := func(id node.ID) proto.Message {
		rv := &message.FindNode{
			Id: id,
		}
		return rv
	}
	return d.findNodeCustom(ctx, id, msgFactory, breakOnResult)
}

// findNodeCustom attempts to locate k closest nodes to a given key (node id).
// If breakOnResult is true the lookup will stop immidiately when a result with
// a matching key is found. That option is good for finding a single node.
// Otherwise the lookup will continue anyway to find k closest nodes. That
// variant is expected when looking for k closest nodes to store certain values
// in the DHT.
func (d *dht) findNodeCustom(ctx context.Context, id node.ID, msgFac messageFactory, breakOnResult bool) ([]node.NodeInfo, error) {
	log.Debugf("findNode %s", id)

	// Register that the lookup was performed to avoid refreshing the bucket
	// that this node falls into during the bootstrap procedure.
	d.rt.PerformedLookup(id)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := newResultsList(id)

	// Initial nodes from kbuckets. Take more than 'a' since some of them
	// may be offline. There is really no real reason why 'k' nodes are
	// picked here - that number is simply significantly larger than 'a'.
	nodes := d.rt.GetClosest(id, k)
	if len(nodes) == 0 {
		log.Debug("findNode buckets empty, aborting")
		return nil, errors.New("Buckets empty")
	}
	for _, nd := range nodes {
		results.Add(d.self.Id, &nd)
	}

	// If the first node is the one that we are looking for: query it, check
	// if it is possible to connect to it and return it if that is the case.
	if breakOnResult {
		if elem := results.list.Front(); elem != nil {
			ndData := elem.Value.(*nodeData)
			if node.CompareId(ndData.Id, id) {
				log.Debug("findNode found in buckets")
				address, err := ndData.GetUnprocessedAddress()
				if err == nil {
					ndInfo := node.NodeInfo{ndData.Id, address.Address}
					address.Processed = true
					_, err := d.netDial(ndInfo)
					if err == nil {
						log.Debug("findNode got a response")
						// Otherwise there will be no results.
						address.Valid = true
						return results.Results(), nil
					}
				}
			}
		}
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
							err := results.Add(msg.Sender.Id, ndInfo)
							log.Debugf("findNode new %s, err %s", ndInfo.Id, err)
						}
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	var prevAllProcessed = false
	for {
		// Send new FindNode messages.
		counterSent := 0
		counterIter := 0

		var allProcessed = true
		for i, entry := range results.Get(k) {
			counterIter = i

			// Stop after sending 'a' messages and await results.
			if counterSent >= a {
				break
			}

			if entry.IsProcessed() {
				continue
			}

			allProcessed = false
			address, err := entry.GetUnprocessedAddress()
			if err != nil {
				continue
			}
			ndInfo := node.NodeInfo{entry.Id, address.Address}
			address.Processed = true
			peer, err := d.netDial(ndInfo)
			if err == nil {
				address.Valid = true
				counterSent++
				msg := msgFac(id)
				log.Debugf("findNode send to %s", ndInfo.Id)
				go peer.SendWithContext(ctx, msg)
			}
		}

		// Already processed all k closest nodes.
		if counterIter >= k && allProcessed {
			log.Debug("findNode counterIter and allProcessed")
			return results.Results(), nil
		}

		// No new results, everything we have has been processed so
		// is no point in waiting for more.
		if prevAllProcessed && allProcessed {
			log.Debug("findNode prevAllProcessed and allProcessed")
			return results.Results(), nil
		}
		prevAllProcessed = allProcessed

		// Await results.
		log.Debug("findNode waiting")
		select {
		case <-ctx.Done():
			return results.Results(), ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}

// addressData stores one of the addresses returned by the nodes during
// the lookup procedure. It is normally linked to a node id by being stored
// in a nodeData structure.
type addressData struct {
	// Network address in the format used by the net package.
	Address string
	// Sources is a list of nodes that sent this address.
	Sources []node.ID
	// Processed is true if the lookup procedure tried connecting to this
	// address.
	Processed bool
	// Valid is true if the lookup procedure tried connecting to this
	// address and succeeded.
	Valid bool
}

// nodeData stores the results of a node lookup for a single node.
type nodeData struct {
	Id        node.ID
	Addresses []*addressData
	Distance  []byte
	lock      sync.Mutex
}

// Insert inserts a new address into the address register. If an address
// already exists it appends the id of the sender to the list of address
// sources.
func (nd *nodeData) Insert(sender node.ID, address string) error {
	nd.lock.Lock()
	defer nd.lock.Unlock()

	for _, addrData := range nd.Addresses {
		if addrData.Address == address {
			for _, id := range addrData.Sources {
				if node.CompareId(sender, id) {
					return nil
				}
			}
			addrData.Sources = append(addrData.Sources, sender)
			return nil
		}
	}
	ad := &addressData{
		Address:   address,
		Sources:   []node.ID{sender},
		Processed: false,
		Valid:     false,
	}
	nd.Addresses = append(nd.Addresses, ad)
	return nil
}

// GetUnprocessedAddress returns an unprocessed address with the highest
// amount of sources (an address that the lookup procedure hasn't checked yet).
func (nd *nodeData) GetUnprocessedAddress() (*addressData, error) {
	nd.lock.Lock()
	defer nd.lock.Unlock()

	maxSources := 0
	i := 0
	for j, addressData := range nd.Addresses {
		l := len(addressData.Sources)
		if !addressData.Processed && l > maxSources {
			maxSources = l
			i = j
		}
	}
	if maxSources > 0 {
		return nd.Addresses[i], nil
	} else {
		return nil, errors.New("Not found")
	}
}

// GetValidAddress returns a confirmed, valid address for this node.
func (nd *nodeData) GetValidAddress() (*addressData, error) {
	nd.lock.Lock()
	defer nd.lock.Unlock()

	for _, addressData := range nd.Addresses {
		if addressData.Valid {
			return addressData, nil
		}
	}
	return nil, errors.New("Not found")
}

// IsProcessed returns true if the address for this node has been found or
// it hasn't been found but there are no more addresses to query.
func (nd *nodeData) IsProcessed() bool {
	if _, err := nd.GetValidAddress(); err == nil {
		return true
	}

	nd.lock.Lock()
	defer nd.lock.Unlock()

	for _, addressData := range nd.Addresses {
		if !addressData.Processed {
			return false
		}
	}
	return true
}

func newResultsList(id node.ID) *resultsList {
	rv := &resultsList{
		id:   id,
		list: list.New(),
	}
	return rv
}

// resultsList stores a list of nodeData entires which are a result of a node
// lookup.
type resultsList struct {
	id   node.ID
	list *list.List
	lock sync.Mutex
}

// Result returns a list of confirmed nodes resulting from the lookup.
func (l *resultsList) Results() []node.NodeInfo {
	l.lock.Lock()
	defer l.lock.Unlock()

	var rv []node.NodeInfo
	for elem := l.list.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(*nodeData)
		addr, err := entry.GetValidAddress()
		if err != nil {
			continue
		}
		info := node.NodeInfo{entry.Id, addr.Address}
		rv = append(rv, info)
	}
	return rv
}

// Add is used to insert lookup results as they arrive. The list of results is
// kept sorted by the distance to the searched id, so the closest node to the
// searched id is always located at the beginning of the list. Errors basically
// mean that the sender node id or the node id nested in the NodeInfo struct
// is simply invalid.
func (l *resultsList) Add(sender node.ID, nd *node.NodeInfo) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	distance, err := node.Distance(l.id, nd.Id)
	if err != nil {
		return err
	}

	newEntry := &nodeData{
		Id:       nd.Id,
		Distance: distance,
	}
	newEntry.Insert(sender, nd.Address)

	// Find an element with a distance bigger or equal to the new one.
	var elem *list.Element = nil
	for elem = l.list.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(*nodeData)
		res, err := utils.Compare(distance, entry.Distance)
		if err != nil {
			return err
		}

		// The entry already exists so we are just going to add a new
		// address to it.
		if res == 0 {
			return entry.Insert(sender, nd.Address)
		}

		// An entry which is further away exists, so we are going to
		// insert a new entry before it.
		if res > 0 {
			l.list.InsertBefore(newEntry, elem)
			return nil
		}
	}

	// The list is either empty or there are no nodes that are further away
	// than the new entry so we can just add it to the end of the list.
	l.list.PushBack(newEntry)
	return nil
}

// Get is used to get lookup results for further queries. It returns first k
// elements from the list.
func (l *resultsList) Get(k int) []*nodeData {
	l.lock.Lock()
	defer l.lock.Unlock()

	var rv []*nodeData
	i := 0
	for elem := l.list.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(*nodeData)
		rv = append(rv, entry)
		i++
		if i > k {
			break
		}
	}
	return rv
}
