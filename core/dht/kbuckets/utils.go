package kbuckets

import (
	"github.com/boreq/starlight/network/node"
)

// randomId creates a random id which has the same length as the provided id
// and in which prefixLen starting bits are identical as in the provided id.
func randomId(self node.ID, prefixLen int) node.ID {
	rv := make([]byte, len(self))
	// Copy the first bits.
outer:
	for i := 0; i < len(rv); i++ {
		for j := 0; j < 8; j++ {
			if 8*i+j >= prefixLen {
				break outer
			}
			mask := byte(1) << byte(7-j)
			bitA := rv[i] & ^mask
			bitB := self[i] & mask
			rv[i] = bitA | bitB
		}
	}

	// Flip the bit that comes after them.
	mask := byte(1) << byte(7-prefixLen%8)
	i := prefixLen / 8
	rv[i] = rv[i] ^ mask
	return rv
}
