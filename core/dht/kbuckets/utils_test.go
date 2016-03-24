package kbuckets

import (
	"github.com/boreq/lainnet/network/node"
	"github.com/boreq/lainnet/utils"
	"testing"
)

func TestRandomId(t *testing.T) {
	const prefixLen = 10
	self := []byte{1, 2, 3, 4}

	// Since this really is random there is a chance that the result is
	// correct by accident.
	for i := 0; i < 100; i++ {
		rand := randomId(self, prefixLen)
		dis, err := node.Distance(self, rand)
		if err != nil {
			t.Fatal(err)
		}
		i := utils.ZerosLen(dis)
		if i != prefixLen {
			t.Fatal("Invalid prefix len", i, prefixLen)
		}
	}
}
