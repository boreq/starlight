package dht

import (
	"container/list"
	"github.com/boreq/starlight/network/node"
	"github.com/boreq/starlight/protocol/message"
	"github.com/boreq/starlight/utils"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"math/rand"
	"sync"
	"time"
)

func (d *dht) FindNode(ctx context.Context, id node.ID) (node.NodeInfo, error) {
	log.Debugf("FindNode %s", id)
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	// Run the lookup procedure.
	nodesLists, err := d.findNode(ctx, id, true)
	if err != nil {
		return node.NodeInfo{}, err
	}
	log.Debugf("FindNode got %d result lists", len(nodesLists))

	// Check if the procedure managed to locate the right node.
	for _, nodes := range nodesLists {
		if len(nodes) > 0 && node.CompareId(nodes[0].Id, id) {
			return nodes[0], nil
		}
	}
	return node.NodeInfo{}, errors.New("node not found")
}

// messageFactory is used to send different messages during the node lookup
// procedure - for example FindPubKey is used instead of FindNode during a key
// lookup, so the calling function has to provide a way of generating that
// message instead of the default one. The provided node ID is the id of the
// searched node.
type messageFactory func(id node.ID) proto.Message

// findNode performs a standard node lookup procedure using the FindNode
// message.
func (d *dht) findNode(ctx context.Context, id node.ID, breakOnResult bool) ([][]node.NodeInfo, error) {
	msgFactory := func(id node.ID) proto.Message {
		rv := &message.FindNode{
			Id: id,
		}
		return rv
	}
	return d.lookup(ctx, id, msgFactory, breakOnResult)
}

// lookup attempts to locate k closest nodes to a given key (node id). The
// procedure is performed over disjoint paths. If breakOnResult is set this
// function will return results the moment a node that we are looking for is
// found in any of the lists.
func (d *dht) lookup(ctx context.Context, id node.ID, msgFac messageFactory, breakOnResult bool) ([][]node.NodeInfo, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Debugf("lookup %s", id)

	// Register that the lookup was performed to avoid refreshing the bucket
	// that this node falls into during the bootstrap procedure.
	d.rt.PerformedLookup(id) // TODO move this somewhere else in this code?

	// Initial nodes from kbuckets. Take more than 'a' since some of them
	// may be offline. There is really no real reason why 'k' nodes are
	// picked here - that number is simply significantly larger than 'a'.
	nodes := d.rt.GetClosest(id, paramK)
	if len(nodes) == 0 {
		return nil, errors.New("buckets returned zero nodes")
	}

	// Split the retuned nodes into 'd' buckets randomly in order to perform
	// a lookup over d disjoint paths.
	var buckets []*resultsList
	for i := 0; i < paramD; i++ {
		buckets = append(buckets, newResultsList(id))
	}
	for i := range nodes {
		j := rand.Intn(i + 1)
		nodes[i], nodes[j] = nodes[j], nodes[i]
	}
	for i := range nodes {
		buckets[i%paramD].Add(d.self.Id, &nodes[i])
	}

	// Start the disjoint lookup procedure for each bucket.
	resultC := make(chan []node.NodeInfo, len(buckets))
	bucketsMutex := &sync.Mutex{}

	startedLookups := 0
	for i := range buckets {
		if len(buckets[i].Get(1)) > 0 {
			startedLookups++
			go d.lookupDisjointPath(ctx, buckets, bucketsMutex, i, id, msgFac, breakOnResult, resultC)
		}
	}

	// Gather results from all disjoint lookup procedures.
	var results [][]node.NodeInfo
	responses := 0
	for {
		select {
		case result := <-resultC:
			responses++
			results = append(results, result)
			// If we are supposed to breakOnResult and the node has
			// been found return immidiately.
			if breakOnResult {
				if len(result) > 0 && node.CompareId(result[0].Id, id) {
					return results, nil
				}
			}
			if responses >= startedLookups {
				return results, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (d *dht) lookupDisjointPath(ctx context.Context, buckets []*resultsList, bucketsMutex *sync.Mutex, bucketI int, id node.ID, msgFac messageFactory, breakOnResult bool, resultC chan<- []node.NodeInfo) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := buckets[bucketI]

	isInOtherBuckets := func(id node.ID) bool {
		for i, bucket := range buckets {
			if i != bucketI {
				if bucket.Contains(id) {
					return true
				}
			}
		}
		return false
	}

	// Handle incoming Nodes messages.
	go func() {
		c, cancel := d.net.Subscribe()
		defer cancel()
		for {
			select {
			case msg := <-c:
				switch pMsg := msg.Message.(type) {
				case *message.Nodes:
					// When a Nodes message is received add
					// every single node to the list of
					// results.
					if results.WasQueried(msg.Sender.Id) {
						for _, nd := range pMsg.Nodes {
							ndInfo := &node.NodeInfo{Id: nd.GetId(), Address: nd.GetAddress()}
							if !node.CompareId(ndInfo.Id, d.self.Id) {
								if !isInOtherBuckets(ndInfo.Id) {
									err := results.Add(msg.Sender.Id, ndInfo)
									log.Debugf("lookupDisjointPath new %s, err %s", ndInfo.Id, err)
								}
							}
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
		counterSent := 0
		counterIter := 0
		allProcessed := true

		// Send new FindNode messages.
		for i, nData := range results.Get(paramK) {
			counterIter = i

			// Stop after sending 'a' messages and await results.
			if counterSent >= paramA {
				break
			}

			// Stop if an address for this entry has been found or
			// it hasn't been found but no more addresses are
			// known.
			if nData.IsProcessed() {
				continue
			}

			allProcessed = false
			address, err := nData.GetUnprocessedAddress()
			if err != nil {
				continue
			}
			if err := nData.MarkAddressProcessed(address.Address); err != nil {
				log.Debugf("error marking address as processed: %s", err)
			}

			ndInfo := node.NodeInfo{Id: nData.Id, Address: address.Address}

			// Send the lookup message to the node.
			p, err := d.netDial(ndInfo)
			if err == nil {
				if breakOnResult {
					if node.CompareId(p.Info().Id, id) {
						address.Valid = true
						resultC <- results.Results()
						return
					}
				}
				counterSent++
				msg := msgFac(id)
				log.Debugf("findNode send to %s", ndInfo.Id)
				go p.SendWithContext(ctx, msg)
			}

			// Check if this node is reachable using this address.
			go func(nData *nodeData) {
				err = d.netCheckOnline(ctx, ndInfo)
				if err == nil {
					if err := nData.MarkAddressValid(address.Address); err != nil {
						log.Debugf("error marking address as valid: %s", err)
					}
				}
			}(nData)
		}

		// Already processed all k closest nodes.
		if counterIter >= paramK && allProcessed {
			log.Debug("findNode counterIter and allProcessed")
			resultC <- results.Results()
			return
		}

		// No new results, everything we have has been processed so
		// is no point in waiting for more.
		if prevAllProcessed && allProcessed {
			log.Debug("findNode prevAllProcessed and allProcessed")
			resultC <- results.Results()
			return
		}
		prevAllProcessed = allProcessed

		// Await results.
		log.Debug("findNode waiting")
		select {
		case <-ctx.Done():
			resultC <- results.Results()
			return
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
	addresses []*addressData
	Distance  []byte
	lock      sync.Mutex
}

// Insert inserts a new address into the address register. If an address
// already exists it appends the id of the sender to the list of address
// sources.
func (nd *nodeData) Insert(sender node.ID, address string) error {
	nd.lock.Lock()
	defer nd.lock.Unlock()

	for _, addrData := range nd.addresses {
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
	nd.addresses = append(nd.addresses, ad)
	return nil
}

// GetUnprocessedAddress returns an unprocessed address with the highest
// amount of sources (an address that the lookup procedure hasn't checked yet).
func (nd *nodeData) GetUnprocessedAddress() (addressData, error) {
	nd.lock.Lock()
	defer nd.lock.Unlock()

	maxSources := 0
	i := 0
	for j, addressData := range nd.addresses {
		l := len(addressData.Sources)
		if !addressData.Processed && l > maxSources {
			maxSources = l
			i = j
		}
	}
	if maxSources > 0 {
		return *nd.addresses[i], nil
	} else {
		return addressData{}, errors.New("Not found")
	}
}

// GetValidAddress returns a confirmed, valid address for this node.
func (nd *nodeData) GetValidAddress() (addressData, error) {
	nd.lock.Lock()
	defer nd.lock.Unlock()

	return nd.getValidAddress()
}

func (nd *nodeData) getValidAddress() (addressData, error) {
	for _, addressData := range nd.addresses {
		if addressData.Valid {
			return *addressData, nil
		}
	}
	return addressData{}, errors.New("Not found")
}

// WasQueried returns true if a message was sent to this node during this
// lookup procedure.
func (nd *nodeData) WasQueried() bool {
	nd.lock.Lock()
	defer nd.lock.Unlock()

	for _, addressData := range nd.addresses {
		if addressData.Processed {
			return true
		}
	}
	return false
}

// IsProcessed returns true if the address for this node has been found or
// it hasn't been found but there are no more addresses to query.
func (nd *nodeData) IsProcessed() bool {
	nd.lock.Lock()
	defer nd.lock.Unlock()

	if _, err := nd.getValidAddress(); err == nil {
		return true
	}

	for _, addressData := range nd.addresses {
		if !addressData.Processed {
			return false
		}
	}
	return true
}

// MarkAddressProcessed marks an address processed. Address can't be returned
// using a pointer and marked as processed directly due to race conditions.
func (nd *nodeData) MarkAddressProcessed(address string) error {
	nd.lock.Lock()
	defer nd.lock.Unlock()

	for _, addressData := range nd.addresses {
		if addressData.Address == address {
			addressData.Processed = true
			return nil
		}
	}
	return errors.New("address not found")
}

// MarkAddressProcessed marks an address processed. Address can't be returned
// using a pointer and marked as processed directly due to race conditions.
func (nd *nodeData) MarkAddressValid(address string) error {
	nd.lock.Lock()
	defer nd.lock.Unlock()

	for _, addressData := range nd.addresses {
		if addressData.Address == address {
			addressData.Valid = true
			return nil
		}
	}
	return errors.New("address not found")
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
		info := node.NodeInfo{Id: entry.Id, Address: addr.Address}
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

// WasQueried is used to check if a message was sent to a node during this
// lookup procedure.
func (l *resultsList) WasQueried(id node.ID) bool {
	l.lock.Lock()
	defer l.lock.Unlock()

	for elem := l.list.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(*nodeData)
		if node.CompareId(entry.Id, id) {
			return entry.WasQueried()
		}
	}
	return false
}

// Contains is used to check if this results list contains a node.
func (l *resultsList) Contains(id node.ID) bool {
	l.lock.Lock()
	defer l.lock.Unlock()

	for elem := l.list.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(*nodeData)
		if node.CompareId(entry.Id, id) {
			return true
		}
	}
	return false
}
