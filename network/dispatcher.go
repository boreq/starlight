package network

import (
	"github.com/boreq/netblog/protocol"
	"golang.org/x/net/context"
	"sync"
)

type CancelFunc func()

// Dispatcher exposes methods which allow to subscribe to messages. Dispatched
// messages are sent to every subscriber through a channel.
type Dispatcher interface {
	// Subscripe returns a channel on which it is possible to receive
	// incoming messages and a CancelFunc which must be called if the
	// calling function no longer wishes to receive messages on a channel.
	Subscribe() (chan IncomingMessage, CancelFunc)

	// Dispatch forwards a message to all channels retrieved using the
	// subscribe method.
	Dispatch(Peer, protocol.Message)
}

func NewDispatcher(ctx context.Context) Dispatcher {
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
		close(sub.C)
		for i := len(d.subs) - 1; i >= 0; i-- {
			if d.subs[i] == sub {
				d.subs = append(d.subs[:i], d.subs[i+1:]...)
			}
		}
	}
	return sub.C, cancel
}

func (d *dispatcher) Dispatch(p Peer, msg protocol.Message) {
	d.lock.Lock()
	defer d.lock.Unlock()

	incMsg := IncomingMessage{
		p.Info(),
		msg,
	}

	log.Printf("Dispatching message from %s", incMsg.Id)

	for _, sub := range d.subs {
		go func() {
			select {
			case <-sub.Ctx.Done():
				return
			case sub.C <- incMsg:
			}
		}()
	}
}
