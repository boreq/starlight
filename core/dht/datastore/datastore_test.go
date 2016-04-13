package datastore

import (
	"testing"
	"time"
)

func TestGetEmpty(t *testing.T) {
	d := New(time.Second)
	key := []byte{0}

	_, err := d.Get(key)
	if err == nil {
		t.Fatal("Should fail - the key doesn't exist")
	}
}

func TestStore(t *testing.T) {
	d := New(time.Second)
	key := []byte{0}

	err := d.Store(key, nil)
	if err != nil {
		t.Fatal("Store failed")
	}

	_, err = d.Get(key)
	if err != nil {
		t.Fatal("Get failed")
	}
}

func TestTimeout(t *testing.T) {
	d := New(time.Second)
	key1 := []byte{0}
	key2 := []byte{1}
	key3 := []byte{2}

	// Insert two keys
	err := d.Store(key1, nil)
	if err != nil {
		t.Fatal("Store failed")
	}

	err = d.Store(key2, nil)
	if err != nil {
		t.Fatal("Store failed")
	}

	<-time.After(2 * time.Second)

	// Insert another one after the timeout to force the deletion of old
	// entries
	err = d.Store(key3, nil)
	if err != nil {
		t.Fatal("Store failed")
	}

	// Check that the old entries are gone and the new one is still present
	_, err = d.Get(key1)
	if err == nil {
		t.Fatal("Get should fail")
	}

	_, err = d.Get(key2)
	if err == nil {
		t.Fatal("Get should fail")
	}

	_, err = d.Get(key3)
	if err != nil {
		t.Fatal("Get should not fail")
	}
}
