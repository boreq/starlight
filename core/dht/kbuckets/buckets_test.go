package kbuckets

import (
	"fmt"
	"testing"
)

func TestBuckets(t *testing.T) {
	selfId := []byte{0x0}

	buckets := New(selfId, 2)

	buckets.Update([]byte{0x8}, "addr1")
	buckets.Update([]byte{0xc}, "addr2")
	buckets.Update([]byte{0xe}, "addr3")

	fmt.Println("Add more distant ones")
	buckets.Update([]byte{0xf0}, "add")

	fmt.Println("Update")
	buckets.Update([]byte{0xe}, "addrnew")

	for _, entry := range buckets.GetClosest(selfId, 2) {
		fmt.Printf("%x %s\n", entry.Id, entry.Address)
	}
}
