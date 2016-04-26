package channelstore

import (
	"github.com/boreq/lainnet/protocol/message"
	"testing"
	"time"
)

func makeMessage(channelId, nodeId []byte) *message.StoreChannel {
	timestamp := time.Now().UTC().Unix()
	msg := &message.StoreChannel{
		ChannelId: channelId,
		NodeId:    nodeId,
		Timestamp: &timestamp,
	}
	return msg
}

func TestGet(t *testing.T) {
	c := New(time.Second)
	entries := c.Get([]byte{0})
	if len(entries) != 0 {
		t.Fatal("Returned entries:", len(entries))
	}
}

func TestStoreGetSimple(t *testing.T) {
	channelKey1 := []byte{0}
	nodeKey1 := []byte{0}
	msg1 := makeMessage(channelKey1, nodeKey1)

	c := New(time.Second)
	err := c.Store(msg1)
	if err != nil {
		t.Fatal(err)
	}

	entries := c.Get(channelKey1)
	if len(entries) != 1 {
		t.Fatal("Returned entries:", len(entries))
	}
}

func TestStoreGet(t *testing.T) {
	channelKey1 := []byte{0}
	nodeKey1 := []byte{0}
	msg1 := makeMessage(channelKey1, nodeKey1)

	channelKey2 := []byte{1}
	nodeKey2 := []byte{1}
	msg2 := makeMessage(channelKey2, nodeKey2)

	c := New(time.Second)

	err := c.Store(msg1)
	if err != nil {
		t.Fatal(err)
	}

	err = c.Store(msg2)
	if err != nil {
		t.Fatal(err)
	}

	entries1 := c.Get(channelKey1)
	if len(entries1) != 1 {
		t.Fatal("Returned entries:", len(entries1))
	}

	entries2 := c.Get(channelKey2)
	if len(entries2) != 1 {
		t.Fatal("Returned entries:", len(entries2))
	}
}

func TestTimeout(t *testing.T) {
	channelKey1 := []byte{0}
	nodeKey1 := []byte{0}
	msg1 := makeMessage(channelKey1, nodeKey1)

	c := New(time.Second)

	err := c.Store(msg1)
	if err != nil {
		t.Fatal(err)
	}

	<-time.After(time.Second)

	channelKey2 := []byte{1}
	nodeKey2 := []byte{1}
	msg2 := makeMessage(channelKey2, nodeKey2)

	err = c.Store(msg2)
	if err != nil {
		t.Fatal(err)
	}

	entries1 := c.Get(channelKey1)
	if len(entries1) != 0 {
		t.Fatal("Returned entries:", len(entries1))
	}

	entries2 := c.Get(channelKey2)
	if len(entries2) != 1 {
		t.Fatal("Returned entries:", len(entries2))
	}
}

func TestTimeoutNoStore(t *testing.T) {
	channelKey1 := []byte{0}
	nodeKey1 := []byte{0}
	msg1 := makeMessage(channelKey1, nodeKey1)

	c := New(time.Second)

	err := c.Store(msg1)
	if err != nil {
		t.Fatal(err)
	}

	<-time.After(time.Second)

	entries1 := c.Get(channelKey1)
	if len(entries1) != 0 {
		t.Fatal("Returned entries:", len(entries1))
	}
}
