package humanizer

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/boreq/friendlyhash"
	"github.com/boreq/starlight/crypto"
	"github.com/boreq/starlight/irc/nickserver"
	"github.com/boreq/starlight/network/node"
	"github.com/boreq/starlight/utils"
	"github.com/pkg/errors"
)

const delimiter = "."
const nickCacheTimeout = 60 * time.Minute
const retryNickUpdateEvery = 1 * time.Minute

var log = utils.GetLogger("humanizer")

func New(ctx context.Context, iden node.Identity, nickServerUrl string, dictionary []string) (*Humanizer, error) {
	friendlyHash, err := friendlyhash.New(dictionary, crypto.KeyDigestLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not create friendlyhash")
	}

	nickServer, err := nickserver.NewNickServerClient(nickServerUrl, &iden)
	if err != nil {
		return nil, errors.Wrap(err, "could not create a nick server client")
	}

	rv := &Humanizer{
		friendlyHash: friendlyHash,
		nickServer:   nickServer,
		nicks:        newNickCacheWithTimeout(nickCacheTimeout),
		ctx:          ctx,
	}
	return rv, nil
}

type Humanizer struct {
	friendlyHash *friendlyhash.FriendlyHash
	nickServer   *nickserver.NickServerClient

	ctx       context.Context
	nick      string
	sentNick  string
	nickMutex sync.Mutex

	nicks      *nickCache
	nicksMutex sync.Mutex
}

func (h *Humanizer) run(ctx context.Context) {
	for {
		select {
		case <-time.After(retryNickUpdateEvery):
			h.updateNick(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (h *Humanizer) updateNick(ctx context.Context) {
	h.nickMutex.Lock()
	defer h.nickMutex.Unlock()
	if h.nick != "" && h.nick != h.sentNick {
		log.Debugf("updateNick: %s", h.nick)
		if err := h.nickServer.Put(h.nick); err != nil {
			log.Debugf("error: %s", err)
		} else {
			h.sentNick = h.nick
		}
	}
}

func (h *Humanizer) HumanizeHost(id node.ID) (string, error) {
	words, err := h.friendlyHash.Humanize(id)
	if err != nil {
		return "", errors.Wrap(err, "could not humanize")
	}
	return strings.Join(words, delimiter), nil
}

func (h *Humanizer) HumanizeNick(id node.ID) (string, error) {
	h.nicksMutex.Lock()
	defer h.nicksMutex.Unlock()

	cachedNick, ok := h.nicks.GetNick(id)
	if !ok {
		nick, err := h.getNick(id)
		if err != nil {
			h.nicks.Put(id, id.String())
			return id.String(), nil
		}
		return nick, nil
	}
	return cachedNick, nil
}

func (h *Humanizer) getNick(id node.ID) (string, error) {
	nick, err := h.nickServer.Get(id)
	if err != nil {
		return "", err
	}

	// Handle colission by refreshing the colliding nick
	existingId, ok := h.nicks.GetId(nick)
	if ok && !node.CompareId(existingId, id) {
		if _, err := h.getNick(existingId); err != nil {
			return "", err
		}
	}

	h.nicks.Put(id, nick)
	return nick, nil
}

func (h *Humanizer) DehumanizeNick(nick string) (node.ID, error) {
	h.nicksMutex.Lock()
	defer h.nicksMutex.Unlock()

	nodeId, ok := h.nicks.GetId(nick)
	if !ok {
		// TODO query the server
		return nil, errors.New("nick not found")
	}

	return nodeId, nil
}

func (h *Humanizer) SetNick(nick string) error {
	h.nickMutex.Lock()
	defer h.nickMutex.Unlock()

	if err := h.nickServer.ValidateNick(nick); err != nil {
		return errors.Wrap(err, "nick invalid")
	}

	h.nick = nick
	go h.updateNick(h.ctx)
	return nil
}

type nickCacheEntry struct {
	Id      node.ID
	Nick    string
	Created time.Time
}

func newNickCacheWithTimeout(timeout time.Duration) *nickCache {
	rv := &nickCache{
		timeout: &timeout,
	}
	return rv
}

func newNickCache() *nickCache {
	rv := &nickCache{
		timeout: nil,
	}
	return rv
}

type nickCache struct {
	entries []nickCacheEntry
	timeout *time.Duration
}

func (n *nickCache) GetNick(id node.ID) (string, bool) {
	for _, entry := range n.entries {
		if node.CompareId(id, entry.Id) {
			if !n.isValid(entry) {
				return "", false
			}
			return entry.Nick, true
		}
	}
	return "", false
}

func (n *nickCache) GetId(nick string) (node.ID, bool) {
	for _, entry := range n.entries {
		if entry.Nick == nick {
			if !n.isValid(entry) {
				return nil, false
			}
			return entry.Id, true
		}
	}
	return nil, false
}

func (n *nickCache) isValid(entry nickCacheEntry) bool {
	if n.timeout == nil {
		return true
	}
	now := time.Now()
	validBefore := entry.Created.Add(*n.timeout)
	return now.Before(validBefore)
}

func (n *nickCache) Put(id node.ID, nick string) {
	// Remove entries with the same nick or id
	for i := len(n.entries) - 1; i >= 0; i-- {
		entry := n.entries[i]
		if entry.Nick == nick || node.CompareId(entry.Id, id) {
			n.entries = append(n.entries[:i], n.entries[i+1:]...)
		}
	}

	// Insert a new entry
	entry := nickCacheEntry{
		Id:      id,
		Nick:    nick,
		Created: time.Now(),
	}
	n.entries = append(n.entries, entry)
}
