// Code generated by protoc-gen-go.
// source: message.proto
// DO NOT EDIT!

/*
Package message is a generated protocol buffer package.

It is generated from these files:
	message.proto

It has these top-level messages:
	Init
*/
package message

import proto "github.com/golang/protobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type Init struct {
	PubKey           []byte `protobuf:"bytes,1,req" json:"PubKey,omitempty"`
	EphemeralPubKey  []byte `protobuf:"bytes,2,req" json:"EphemeralPubKey,omitempty"`
	XXX_unrecognized []byte `json:"-"`
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

func (m *Init) GetEphemeralPubKey() []byte {
	if m != nil {
		return m.EphemeralPubKey
	}
	return nil
}