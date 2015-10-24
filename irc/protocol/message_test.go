package protocol

import "testing"

func TestMarshalParams(t *testing.T) {
	msg := &Message{
		Prefix:  "prefix",
		Command: "command",
		Params:  []string{"p1", "p2"},
	}
	s := msg.Marshal()
	if s != ":prefix command p1 p2" {
		t.Fatal(s)
	}
}

func TestMarshalSpace(t *testing.T) {
	msg := &Message{
		Command: "command",
		Params:  []string{"p1", "p2 with space"},
	}
	s := msg.Marshal()
	if s != "command p1 :p2 with space" {
		t.Fatal(s)
	}
}

func TestUnmarshal(t *testing.T) {
	text := ":irc.example.com 251 botnet_test :There are 185 users on 25 servers"
	msg := UnmarshalMessage(text)
	if msg.Prefix != "irc.example.com" {
		t.Fatal(msg)
	}
	if msg.Command != "251" {
		t.Fatal(msg)
	}
	if msg.Params[0] != "botnet_test" {
		t.Fatal(msg)
	}
	if msg.Params[1] != "There are 185 users on 25 servers" {
		t.Fatal(msg)
	}
}
