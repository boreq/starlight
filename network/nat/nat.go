package nat

import (
	"context"
	"sync"
	"time"

	"github.com/boreq/starlight/utils"
	libnat "github.com/libp2p/go-libp2p-nat"
	"github.com/pkg/errors"
)

var log = utils.GetLogger("network/nat")

func New(ctx context.Context, internalPort int) (*NAT, error) {
	rv := &NAT{
		ctx:          ctx,
		internalPort: internalPort,
	}
	go rv.run()
	return rv, nil
}

type NAT struct {
	ctx          context.Context
	nat          *libnat.NAT
	mapping      libnat.Mapping
	internalPort int
	mutex        sync.Mutex
}

func (n *NAT) run() {
	for {
		if err := n.refresh(); err != nil {
			log.Debugf("refresh failed: %s", err)
		}

		select {
		case <-n.ctx.Done():
			return
		case <-time.After(60 * time.Second):
		}
	}
}

func (n *NAT) refresh() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.nat == nil {
		log.Debug("running DiscoverNAT")
		nat, err := libnat.DiscoverNAT(n.ctx)
		if err != nil {
			return errors.Wrap(err, "nat discovery failed")
		}
		n.nat = nat
	}

	if n.nat != nil && n.mapping == nil {
		log.Debug("running NewMapping")
		mapping, err := n.nat.NewMapping("tcp", n.internalPort)
		if err != nil {
			return errors.Wrap(err, "nat mapping failed")
		}
		n.mapping = mapping
		log.Debugf("external port: %d", n.mapping.ExternalPort())
	}

	return nil
}

func (n *NAT) GetAddress() (string, error) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.mapping != nil {
		if addr, err := n.mapping.ExternalAddr(); err != nil {
			return "", err
		} else {
			return addr.String(), nil
		}
	}
	return "", errors.New("mapping is null")
}
