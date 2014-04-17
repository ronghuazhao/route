// Code generated by protoc-gen-go.
// source: global.proto
// DO NOT EDIT!

package interfaces

import proto "code.google.com/p/goprotobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type DO int32

const (
	DO_UPDATE DO = 0
	DO_DELETE DO = 1
)

var DO_name = map[int32]string{
	0: "UPDATE",
	1: "DELETE",
}
var DO_value = map[string]int32{
	"UPDATE": 0,
	"DELETE": 1,
}

func (x DO) Enum() *DO {
	p := new(DO)
	*p = x
	return p
}
func (x DO) String() string {
	return proto.EnumName(DO_name, int32(x))
}
func (x *DO) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(DO_value, data, "DO")
	if err != nil {
		return err
	}
	*x = DO(value)
	return nil
}

func init() {
	proto.RegisterEnum("interfaces.DO", DO_name, DO_value)
}