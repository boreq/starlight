// Code generated by protoc-gen-go.
// source: message.proto
// DO NOT EDIT!

/*
Package message is a generated protocol buffer package.

It is generated from these files:
	message.proto

It has these top-level messages:
	Init
	Handshake
	ConfirmHandshake
	Identity
	Ping
	Pong
	FindNode
	Nodes
	PrivateMessage
	StorePubKey
*/
package message

import proto "github.com/golang/protobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type Init struct {
	PubKey           []byte  `protobuf:"bytes,1,req" json:"PubKey,omitempty"`
	Nonce            []byte  `protobuf:"bytes,2,req" json:"Nonce,omitempty"`
	SupportedCurves  *string `protobuf:"bytes,3,req" json:"SupportedCurves,omitempty"`
	SupportedHashes  *string `protobuf:"bytes,4,req" json:"SupportedHashes,omitempty"`
	SupportedCiphers *string `protobuf:"bytes,5,req" json:"SupportedCiphers,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *Init) Reset()         { *m = Init{} }
func (m *Init) String() string { return proto.CompactTextString(m) }
func (*Init) ProtoMessage()    {}

func (m *Init) GetPubKey() []byte {
	if m != nil {
		return m.PubKey
	}
	return nil
}

func (m *Init) GetNonce() []byte {
	if m != nil {
		return m.Nonce
	}
	return nil
}

func (m *Init) GetSupportedCurves() string {
	if m != nil && m.SupportedCurves != nil {
		return *m.SupportedCurves
	}
	return ""
}

func (m *Init) GetSupportedHashes() string {
	if m != nil && m.SupportedHashes != nil {
		return *m.SupportedHashes
	}
	return ""
}

func (m *Init) GetSupportedCiphers() string {
	if m != nil && m.SupportedCiphers != nil {
		return *m.SupportedCiphers
	}
	return ""
}

type Handshake struct {
	EphemeralPubKey  []byte `protobuf:"bytes,1,req" json:"EphemeralPubKey,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *Handshake) Reset()         { *m = Handshake{} }
func (m *Handshake) String() string { return proto.CompactTextString(m) }
func (*Handshake) ProtoMessage()    {}

func (m *Handshake) GetEphemeralPubKey() []byte {
	if m != nil {
		return m.EphemeralPubKey
	}
	return nil
}

type ConfirmHandshake struct {
	Nonce            []byte `protobuf:"bytes,1,req" json:"Nonce,omitempty"`
	Signature        []byte `protobuf:"bytes,2,req" json:"Signature,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *ConfirmHandshake) Reset()         { *m = ConfirmHandshake{} }
func (m *ConfirmHandshake) String() string { return proto.CompactTextString(m) }
func (*ConfirmHandshake) ProtoMessage()    {}

func (m *ConfirmHandshake) GetNonce() []byte {
	if m != nil {
		return m.Nonce
	}
	return nil
}

func (m *ConfirmHandshake) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

type Identity struct {
	// Address the local node is listening on.
	ListenAddress *string `protobuf:"bytes,1,req" json:"ListenAddress,omitempty"`
	// Apparent address of the other side of the connection.
	ConnectionAddress *string `protobuf:"bytes,2,req" json:"ConnectionAddress,omitempty"`
	XXX_unrecognized  []byte  `json:"-"`
}

func (m *Identity) Reset()         { *m = Identity{} }
func (m *Identity) String() string { return proto.CompactTextString(m) }
func (*Identity) ProtoMessage()    {}

func (m *Identity) GetListenAddress() string {
	if m != nil && m.ListenAddress != nil {
		return *m.ListenAddress
	}
	return ""
}

func (m *Identity) GetConnectionAddress() string {
	if m != nil && m.ConnectionAddress != nil {
		return *m.ConnectionAddress
	}
	return ""
}

type Ping struct {
	Random           *uint32 `protobuf:"varint,1,req" json:"Random,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *Ping) Reset()         { *m = Ping{} }
func (m *Ping) String() string { return proto.CompactTextString(m) }
func (*Ping) ProtoMessage()    {}

func (m *Ping) GetRandom() uint32 {
	if m != nil && m.Random != nil {
		return *m.Random
	}
	return 0
}

type Pong struct {
	Random           *uint32 `protobuf:"varint,1,req" json:"Random,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *Pong) Reset()         { *m = Pong{} }
func (m *Pong) String() string { return proto.CompactTextString(m) }
func (*Pong) ProtoMessage()    {}

func (m *Pong) GetRandom() uint32 {
	if m != nil && m.Random != nil {
		return *m.Random
	}
	return 0
}

type FindNode struct {
	Id               []byte `protobuf:"bytes,1,req" json:"Id,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *FindNode) Reset()         { *m = FindNode{} }
func (m *FindNode) String() string { return proto.CompactTextString(m) }
func (*FindNode) ProtoMessage()    {}

func (m *FindNode) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

type Nodes struct {
	Nodes            []*Nodes_NodeInfo `protobuf:"bytes,1,rep" json:"Nodes,omitempty"`
	XXX_unrecognized []byte            `json:"-"`
}

func (m *Nodes) Reset()         { *m = Nodes{} }
func (m *Nodes) String() string { return proto.CompactTextString(m) }
func (*Nodes) ProtoMessage()    {}

func (m *Nodes) GetNodes() []*Nodes_NodeInfo {
	if m != nil {
		return m.Nodes
	}
	return nil
}

type Nodes_NodeInfo struct {
	Id               []byte  `protobuf:"bytes,1,req" json:"Id,omitempty"`
	Address          *string `protobuf:"bytes,2,req" json:"Address,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *Nodes_NodeInfo) Reset()         { *m = Nodes_NodeInfo{} }
func (m *Nodes_NodeInfo) String() string { return proto.CompactTextString(m) }
func (*Nodes_NodeInfo) ProtoMessage()    {}

func (m *Nodes_NodeInfo) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *Nodes_NodeInfo) GetAddress() string {
	if m != nil && m.Address != nil {
		return *m.Address
	}
	return ""
}

type PrivateMessage struct {
	Text             *string `protobuf:"bytes,1,req" json:"Text,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *PrivateMessage) Reset()         { *m = PrivateMessage{} }
func (m *PrivateMessage) String() string { return proto.CompactTextString(m) }
func (*PrivateMessage) ProtoMessage()    {}

func (m *PrivateMessage) GetText() string {
	if m != nil && m.Text != nil {
		return *m.Text
	}
	return ""
}

type StorePubKey struct {
	Key              []byte `protobuf:"bytes,1,req" json:"Key,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *StorePubKey) Reset()         { *m = StorePubKey{} }
func (m *StorePubKey) String() string { return proto.CompactTextString(m) }
func (*StorePubKey) ProtoMessage()    {}

func (m *StorePubKey) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}
