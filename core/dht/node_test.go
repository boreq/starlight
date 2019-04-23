package dht

import "testing"

// TestGetUnprocessedAddress makes sure that GetUnprocessedAddress returns
// the address with the highest amount of votes and doesn't return processed
// addresses.
func TestGetUnprocessedAddress(t *testing.T) {
	id := []byte{0}
	nd := &nodeData{Id: id}

	err := nd.Insert([]byte{1}, "addr1")
	if err != nil {
		t.Fatal(err)
	}

	err = nd.Insert([]byte{1}, "addr2")
	if err != nil {
		t.Fatal(err)
	}

	err = nd.Insert([]byte{2}, "addr2")
	if err != nil {
		t.Fatal(err)
	}

	data1, err := nd.GetUnprocessedAddress()
	if err != nil {
		t.Fatal(err)
	}
	if data1.Address != "addr2" {
		t.Fatal("addr2 had more votes")
	}

	if err := nd.MarkAddressProcessed(data1.Address); err != nil {
		t.Fatalf("error marking as processed %s", err)
	}

	data2, err := nd.GetUnprocessedAddress()
	if err != nil {
		t.Fatal(err)
	}
	if data1.Address == data2.Address {
		t.Fatal("Returned the same address")
	}

}
