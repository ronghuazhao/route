// Code generated by protoc-gen-go.
// source: auth.proto
// DO NOT EDIT!

/*
Package interfaces is a generated protocol buffer package.

It is generated from these files:
	auth.proto
	global.proto
	route.proto

It has these top-level messages:
	Auth
*/
package interfaces

import proto "code.google.com/p/goprotobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type Auth struct {
	Do               *DO     `protobuf:"varint,1,req,name=do,enum=interfaces.DO" json:"do,omitempty"`
	Email            *string `protobuf:"bytes,2,req,name=email" json:"email,omitempty"`
	PublicKey        *string `protobuf:"bytes,3,req,name=public_key" json:"public_key,omitempty"`
	PrivateKey       *string `protobuf:"bytes,4,req,name=private_key" json:"private_key,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *Auth) Reset()         { *m = Auth{} }
func (m *Auth) String() string { return proto.CompactTextString(m) }
func (*Auth) ProtoMessage()    {}

func (m *Auth) GetDo() DO {
	if m != nil && m.Do != nil {
		return *m.Do
	}
	return DO_UPDATE
}

func (m *Auth) GetEmail() string {
	if m != nil && m.Email != nil {
		return *m.Email
	}
	return ""
}

func (m *Auth) GetPublicKey() string {
	if m != nil && m.PublicKey != nil {
		return *m.PublicKey
	}
	return ""
}

func (m *Auth) GetPrivateKey() string {
	if m != nil && m.PrivateKey != nil {
		return *m.PrivateKey
	}
	return ""
}

func init() {
}
