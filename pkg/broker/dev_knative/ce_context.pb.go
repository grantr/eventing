// Code generated by protoc-gen-go. DO NOT EDIT.
// source: ce_context.proto

package dev_knative

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type CloudEventContext struct {
	Specversion          string   `protobuf:"bytes,1,opt,name=specversion,proto3" json:"specversion,omitempty"`
	Type                 string   `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	Source               string   `protobuf:"bytes,3,opt,name=source,proto3" json:"source,omitempty"`
	Schemaurl            string   `protobuf:"bytes,4,opt,name=schemaurl,proto3" json:"schemaurl,omitempty"`
	Datamediatype        string   `protobuf:"bytes,5,opt,name=datamediatype,proto3" json:"datamediatype,omitempty"`
	Datacontenttype      string   `protobuf:"bytes,6,opt,name=datacontenttype,proto3" json:"datacontenttype,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CloudEventContext) Reset()         { *m = CloudEventContext{} }
func (m *CloudEventContext) String() string { return proto.CompactTextString(m) }
func (*CloudEventContext) ProtoMessage()    {}
func (*CloudEventContext) Descriptor() ([]byte, []int) {
	return fileDescriptor_fc676048f2e074ad, []int{0}
}

func (m *CloudEventContext) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CloudEventContext.Unmarshal(m, b)
}
func (m *CloudEventContext) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CloudEventContext.Marshal(b, m, deterministic)
}
func (m *CloudEventContext) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CloudEventContext.Merge(m, src)
}
func (m *CloudEventContext) XXX_Size() int {
	return xxx_messageInfo_CloudEventContext.Size(m)
}
func (m *CloudEventContext) XXX_DiscardUnknown() {
	xxx_messageInfo_CloudEventContext.DiscardUnknown(m)
}

var xxx_messageInfo_CloudEventContext proto.InternalMessageInfo

func (m *CloudEventContext) GetSpecversion() string {
	if m != nil {
		return m.Specversion
	}
	return ""
}

func (m *CloudEventContext) GetType() string {
	if m != nil {
		return m.Type
	}
	return ""
}

func (m *CloudEventContext) GetSource() string {
	if m != nil {
		return m.Source
	}
	return ""
}

func (m *CloudEventContext) GetSchemaurl() string {
	if m != nil {
		return m.Schemaurl
	}
	return ""
}

func (m *CloudEventContext) GetDatamediatype() string {
	if m != nil {
		return m.Datamediatype
	}
	return ""
}

func (m *CloudEventContext) GetDatacontenttype() string {
	if m != nil {
		return m.Datacontenttype
	}
	return ""
}

func init() {
	proto.RegisterType((*CloudEventContext)(nil), "dev.knative.CloudEventContext")
}

func init() { proto.RegisterFile("ce_context.proto", fileDescriptor_fc676048f2e074ad) }

var fileDescriptor_fc676048f2e074ad = []byte{
	// 188 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x48, 0x4e, 0x8d, 0x4f,
	0xce, 0xcf, 0x2b, 0x49, 0xad, 0x28, 0xd1, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x4e, 0x49,
	0x2d, 0xd3, 0xcb, 0xce, 0x4b, 0x2c, 0xc9, 0x2c, 0x4b, 0x55, 0x3a, 0xcf, 0xc8, 0x25, 0xe8, 0x9c,
	0x93, 0x5f, 0x9a, 0xe2, 0x5a, 0x96, 0x9a, 0x57, 0xe2, 0x0c, 0x51, 0x28, 0xa4, 0xc0, 0xc5, 0x5d,
	0x5c, 0x90, 0x9a, 0x5c, 0x96, 0x5a, 0x54, 0x9c, 0x99, 0x9f, 0x27, 0xc1, 0xa8, 0xc0, 0xa8, 0xc1,
	0x19, 0x84, 0x2c, 0x24, 0x24, 0xc4, 0xc5, 0x52, 0x52, 0x59, 0x90, 0x2a, 0xc1, 0x04, 0x96, 0x02,
	0xb3, 0x85, 0xc4, 0xb8, 0xd8, 0x8a, 0xf3, 0x4b, 0x8b, 0x92, 0x53, 0x25, 0x98, 0xc1, 0xa2, 0x50,
	0x9e, 0x90, 0x0c, 0x17, 0x67, 0x71, 0x72, 0x46, 0x6a, 0x6e, 0x62, 0x69, 0x51, 0x8e, 0x04, 0x0b,
	0x58, 0x0a, 0x21, 0x20, 0xa4, 0xc2, 0xc5, 0x9b, 0x92, 0x58, 0x92, 0x98, 0x9b, 0x9a, 0x92, 0x99,
	0x08, 0x36, 0x92, 0x15, 0xac, 0x02, 0x55, 0x50, 0x48, 0x83, 0x8b, 0x1f, 0x24, 0x00, 0xf6, 0x49,
	0x5e, 0x09, 0x58, 0x1d, 0x1b, 0x58, 0x1d, 0xba, 0x70, 0x12, 0x1b, 0xd8, 0x97, 0xc6, 0x80, 0x00,
	0x00, 0x00, 0xff, 0xff, 0x9f, 0x0a, 0x80, 0x22, 0xf9, 0x00, 0x00, 0x00,
}
