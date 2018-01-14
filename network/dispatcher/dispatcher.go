package dispatcher

import (
	"github.com/boreq/starlight/network/node"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"sync"
)

// New creates a new Dispatcher. The provided context is used as a parent
// context for contexts in all subscriptions so if this context is closed every
// subscription will be closed as well.
func New(ctx context.Context) Dispatcher {
	rw := &dispatcher{
		ctx: ctx,
	}
	return rw
}

type subscription struct {
	C   chan IncomingMessage
	Ctx context.Context
}

type dispatcher struct {
	subs []*subscription
	lock sync.Mutex
	ctx  context.Context
}

func (d *dispatcher) Subscribe() (chan IncomingMessage, CancelFunc) {
	d.lock.Lock()
	defer d.lock.Unlock()

	ctx, ctxCancel := context.WithCancel(d.ctx)
	sub := &subscription{
		C:   make(chan IncomingMessage),
		Ctx: ctx,
	}
	d.subs = append(d.subs, sub)
	cancel := func() {
		d.lock.Lock()
		defer d.lock.Unlock()
		ctxCancel()
		for i := len(d.subs) - 1; i >= 0; i-- {
			if d.subs[i] == sub {
				d.subs = append(d.subs[:i], d.subs[i+1:]...)
			}
		}
	}
	return sub.C, cancel
}

func (d *dispatcher) Dispatch(node node.NodeInfo, msg proto.Message) {
	d.lock.Lock()
	defer d.lock.Unlock()

	incMsg := IncomingMessage{
		node,
		msg,
	}

	for _, sub := range d.subs {
		go dispatch(*sub, incMsg)
	}
}

func dispatch(sub subscription, msg IncomingMessage) {
	select {
	case <-sub.Ctx.Done():
		return
	case sub.C <- msg:
	}
}
