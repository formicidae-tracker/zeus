// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.12.4
// source: zeus_service.proto

package zeuspb

import (
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Empty struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *Empty) Reset() {
	*x = Empty{}
	if protoimpl.UnsafeEnabled {
		mi := &file_zeus_service_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Empty) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Empty) ProtoMessage() {}

func (x *Empty) ProtoReflect() protoreflect.Message {
	mi := &file_zeus_service_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Empty.ProtoReflect.Descriptor instead.
func (*Empty) Descriptor() ([]byte, []int) {
	return file_zeus_service_proto_rawDescGZIP(), []int{0}
}

type Target struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name         string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Temperature  *float32 `protobuf:"fixed32,2,opt,name=temperature,proto3,oneof" json:"temperature,omitempty"`
	Humidity     *float32 `protobuf:"fixed32,3,opt,name=humidity,proto3,oneof" json:"humidity,omitempty"`
	Wind         *float32 `protobuf:"fixed32,4,opt,name=wind,proto3,oneof" json:"wind,omitempty"`
	VisibleLight *float32 `protobuf:"fixed32,5,opt,name=visible_light,json=visibleLight,proto3,oneof" json:"visible_light,omitempty"`
	UvLight      *float32 `protobuf:"fixed32,6,opt,name=uv_light,json=uvLight,proto3,oneof" json:"uv_light,omitempty"`
}

func (x *Target) Reset() {
	*x = Target{}
	if protoimpl.UnsafeEnabled {
		mi := &file_zeus_service_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Target) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Target) ProtoMessage() {}

func (x *Target) ProtoReflect() protoreflect.Message {
	mi := &file_zeus_service_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Target.ProtoReflect.Descriptor instead.
func (*Target) Descriptor() ([]byte, []int) {
	return file_zeus_service_proto_rawDescGZIP(), []int{1}
}

func (x *Target) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Target) GetTemperature() float32 {
	if x != nil && x.Temperature != nil {
		return *x.Temperature
	}
	return 0
}

func (x *Target) GetHumidity() float32 {
	if x != nil && x.Humidity != nil {
		return *x.Humidity
	}
	return 0
}

func (x *Target) GetWind() float32 {
	if x != nil && x.Wind != nil {
		return *x.Wind
	}
	return 0
}

func (x *Target) GetVisibleLight() float32 {
	if x != nil && x.VisibleLight != nil {
		return *x.VisibleLight
	}
	return 0
}

func (x *Target) GetUvLight() float32 {
	if x != nil && x.UvLight != nil {
		return *x.UvLight
	}
	return 0
}

type StartRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	SeasonFile string `protobuf:"bytes,1,opt,name=season_file,json=seasonFile,proto3" json:"season_file,omitempty"`
	Version    string `protobuf:"bytes,2,opt,name=version,proto3" json:"version,omitempty"`
}

func (x *StartRequest) Reset() {
	*x = StartRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_zeus_service_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StartRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StartRequest) ProtoMessage() {}

func (x *StartRequest) ProtoReflect() protoreflect.Message {
	mi := &file_zeus_service_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StartRequest.ProtoReflect.Descriptor instead.
func (*StartRequest) Descriptor() ([]byte, []int) {
	return file_zeus_service_proto_rawDescGZIP(), []int{2}
}

func (x *StartRequest) GetSeasonFile() string {
	if x != nil {
		return x.SeasonFile
	}
	return ""
}

func (x *StartRequest) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

type ZoneStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name        string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Temperature *float32 `protobuf:"fixed32,2,opt,name=temperature,proto3,oneof" json:"temperature,omitempty"`
	Humidity    *float32 `protobuf:"fixed32,3,opt,name=humidity,proto3,oneof" json:"humidity,omitempty"`
	Target      *Target  `protobuf:"bytes,4,opt,name=target,proto3" json:"target,omitempty"`
}

func (x *ZoneStatus) Reset() {
	*x = ZoneStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_zeus_service_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ZoneStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ZoneStatus) ProtoMessage() {}

func (x *ZoneStatus) ProtoReflect() protoreflect.Message {
	mi := &file_zeus_service_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ZoneStatus.ProtoReflect.Descriptor instead.
func (*ZoneStatus) Descriptor() ([]byte, []int) {
	return file_zeus_service_proto_rawDescGZIP(), []int{3}
}

func (x *ZoneStatus) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ZoneStatus) GetTemperature() float32 {
	if x != nil && x.Temperature != nil {
		return *x.Temperature
	}
	return 0
}

func (x *ZoneStatus) GetHumidity() float32 {
	if x != nil && x.Humidity != nil {
		return *x.Humidity
	}
	return 0
}

func (x *ZoneStatus) GetTarget() *Target {
	if x != nil {
		return x.Target
	}
	return nil
}

type Status struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Running bool                 `protobuf:"varint,1,opt,name=running,proto3" json:"running,omitempty"`
	Since   *timestamp.Timestamp `protobuf:"bytes,2,opt,name=since,proto3" json:"since,omitempty"`
	Version string               `protobuf:"bytes,3,opt,name=version,proto3" json:"version,omitempty"`
	Zones   []*ZoneStatus        `protobuf:"bytes,4,rep,name=zones,proto3" json:"zones,omitempty"`
}

func (x *Status) Reset() {
	*x = Status{}
	if protoimpl.UnsafeEnabled {
		mi := &file_zeus_service_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Status) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Status) ProtoMessage() {}

func (x *Status) ProtoReflect() protoreflect.Message {
	mi := &file_zeus_service_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Status.ProtoReflect.Descriptor instead.
func (*Status) Descriptor() ([]byte, []int) {
	return file_zeus_service_proto_rawDescGZIP(), []int{4}
}

func (x *Status) GetRunning() bool {
	if x != nil {
		return x.Running
	}
	return false
}

func (x *Status) GetSince() *timestamp.Timestamp {
	if x != nil {
		return x.Since
	}
	return nil
}

func (x *Status) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *Status) GetZones() []*ZoneStatus {
	if x != nil {
		return x.Zones
	}
	return nil
}

var File_zeus_service_proto protoreflect.FileDescriptor

var file_zeus_service_proto_rawDesc = []byte{
	0x0a, 0x12, 0x7a, 0x65, 0x75, 0x73, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0f, 0x66, 0x6f, 0x72, 0x74, 0x2e, 0x7a, 0x65, 0x75, 0x73, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x07, 0x0a, 0x05, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22,
	0x8c, 0x02, 0x0a, 0x06, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x25,
	0x0a, 0x0b, 0x74, 0x65, 0x6d, 0x70, 0x65, 0x72, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x02, 0x48, 0x00, 0x52, 0x0b, 0x74, 0x65, 0x6d, 0x70, 0x65, 0x72, 0x61, 0x74, 0x75,
	0x72, 0x65, 0x88, 0x01, 0x01, 0x12, 0x1f, 0x0a, 0x08, 0x68, 0x75, 0x6d, 0x69, 0x64, 0x69, 0x74,
	0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x02, 0x48, 0x01, 0x52, 0x08, 0x68, 0x75, 0x6d, 0x69, 0x64,
	0x69, 0x74, 0x79, 0x88, 0x01, 0x01, 0x12, 0x17, 0x0a, 0x04, 0x77, 0x69, 0x6e, 0x64, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x02, 0x48, 0x02, 0x52, 0x04, 0x77, 0x69, 0x6e, 0x64, 0x88, 0x01, 0x01, 0x12,
	0x28, 0x0a, 0x0d, 0x76, 0x69, 0x73, 0x69, 0x62, 0x6c, 0x65, 0x5f, 0x6c, 0x69, 0x67, 0x68, 0x74,
	0x18, 0x05, 0x20, 0x01, 0x28, 0x02, 0x48, 0x03, 0x52, 0x0c, 0x76, 0x69, 0x73, 0x69, 0x62, 0x6c,
	0x65, 0x4c, 0x69, 0x67, 0x68, 0x74, 0x88, 0x01, 0x01, 0x12, 0x1e, 0x0a, 0x08, 0x75, 0x76, 0x5f,
	0x6c, 0x69, 0x67, 0x68, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x02, 0x48, 0x04, 0x52, 0x07, 0x75,
	0x76, 0x4c, 0x69, 0x67, 0x68, 0x74, 0x88, 0x01, 0x01, 0x42, 0x0e, 0x0a, 0x0c, 0x5f, 0x74, 0x65,
	0x6d, 0x70, 0x65, 0x72, 0x61, 0x74, 0x75, 0x72, 0x65, 0x42, 0x0b, 0x0a, 0x09, 0x5f, 0x68, 0x75,
	0x6d, 0x69, 0x64, 0x69, 0x74, 0x79, 0x42, 0x07, 0x0a, 0x05, 0x5f, 0x77, 0x69, 0x6e, 0x64, 0x42,
	0x10, 0x0a, 0x0e, 0x5f, 0x76, 0x69, 0x73, 0x69, 0x62, 0x6c, 0x65, 0x5f, 0x6c, 0x69, 0x67, 0x68,
	0x74, 0x42, 0x0b, 0x0a, 0x09, 0x5f, 0x75, 0x76, 0x5f, 0x6c, 0x69, 0x67, 0x68, 0x74, 0x22, 0x49,
	0x0a, 0x0c, 0x53, 0x74, 0x61, 0x72, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1f,
	0x0a, 0x0b, 0x73, 0x65, 0x61, 0x73, 0x6f, 0x6e, 0x5f, 0x66, 0x69, 0x6c, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0a, 0x73, 0x65, 0x61, 0x73, 0x6f, 0x6e, 0x46, 0x69, 0x6c, 0x65, 0x12,
	0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x22, 0xb6, 0x01, 0x0a, 0x0a, 0x5a, 0x6f,
	0x6e, 0x65, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x25, 0x0a, 0x0b,
	0x74, 0x65, 0x6d, 0x70, 0x65, 0x72, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x02, 0x48, 0x00, 0x52, 0x0b, 0x74, 0x65, 0x6d, 0x70, 0x65, 0x72, 0x61, 0x74, 0x75, 0x72, 0x65,
	0x88, 0x01, 0x01, 0x12, 0x1f, 0x0a, 0x08, 0x68, 0x75, 0x6d, 0x69, 0x64, 0x69, 0x74, 0x79, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x02, 0x48, 0x01, 0x52, 0x08, 0x68, 0x75, 0x6d, 0x69, 0x64, 0x69, 0x74,
	0x79, 0x88, 0x01, 0x01, 0x12, 0x2f, 0x0a, 0x06, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x66, 0x6f, 0x72, 0x74, 0x2e, 0x7a, 0x65, 0x75, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x52, 0x06, 0x74,
	0x61, 0x72, 0x67, 0x65, 0x74, 0x42, 0x0e, 0x0a, 0x0c, 0x5f, 0x74, 0x65, 0x6d, 0x70, 0x65, 0x72,
	0x61, 0x74, 0x75, 0x72, 0x65, 0x42, 0x0b, 0x0a, 0x09, 0x5f, 0x68, 0x75, 0x6d, 0x69, 0x64, 0x69,
	0x74, 0x79, 0x22, 0xa1, 0x01, 0x0a, 0x06, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x18, 0x0a,
	0x07, 0x72, 0x75, 0x6e, 0x6e, 0x69, 0x6e, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07,
	0x72, 0x75, 0x6e, 0x6e, 0x69, 0x6e, 0x67, 0x12, 0x30, 0x0a, 0x05, 0x73, 0x69, 0x6e, 0x63, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61,
	0x6d, 0x70, 0x52, 0x05, 0x73, 0x69, 0x6e, 0x63, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72,
	0x73, 0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73,
	0x69, 0x6f, 0x6e, 0x12, 0x31, 0x0a, 0x05, 0x7a, 0x6f, 0x6e, 0x65, 0x73, 0x18, 0x04, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x66, 0x6f, 0x72, 0x74, 0x2e, 0x7a, 0x65, 0x75, 0x73, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x5a, 0x6f, 0x6e, 0x65, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52,
	0x05, 0x7a, 0x6f, 0x6e, 0x65, 0x73, 0x32, 0xca, 0x01, 0x0a, 0x04, 0x5a, 0x65, 0x75, 0x73, 0x12,
	0x45, 0x0a, 0x0c, 0x53, 0x74, 0x61, 0x72, 0x74, 0x43, 0x6c, 0x69, 0x6d, 0x61, 0x74, 0x65, 0x12,
	0x1d, 0x2e, 0x66, 0x6f, 0x72, 0x74, 0x2e, 0x7a, 0x65, 0x75, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2e, 0x53, 0x74, 0x61, 0x72, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16,
	0x2e, 0x66, 0x6f, 0x72, 0x74, 0x2e, 0x7a, 0x65, 0x75, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x3c, 0x0a, 0x09, 0x47, 0x65, 0x74, 0x53, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x12, 0x16, 0x2e, 0x66, 0x6f, 0x72, 0x74, 0x2e, 0x7a, 0x65, 0x75, 0x73, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x17, 0x2e, 0x66, 0x6f,
	0x72, 0x74, 0x2e, 0x7a, 0x65, 0x75, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x53, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x12, 0x3d, 0x0a, 0x0b, 0x53, 0x74, 0x6f, 0x70, 0x43, 0x6c, 0x69, 0x6d,
	0x61, 0x74, 0x65, 0x12, 0x16, 0x2e, 0x66, 0x6f, 0x72, 0x74, 0x2e, 0x7a, 0x65, 0x75, 0x73, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x16, 0x2e, 0x66, 0x6f,
	0x72, 0x74, 0x2e, 0x7a, 0x65, 0x75, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x45, 0x6d,
	0x70, 0x74, 0x79, 0x42, 0x0a, 0x5a, 0x08, 0x2e, 0x3b, 0x7a, 0x65, 0x75, 0x73, 0x70, 0x62, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_zeus_service_proto_rawDescOnce sync.Once
	file_zeus_service_proto_rawDescData = file_zeus_service_proto_rawDesc
)

func file_zeus_service_proto_rawDescGZIP() []byte {
	file_zeus_service_proto_rawDescOnce.Do(func() {
		file_zeus_service_proto_rawDescData = protoimpl.X.CompressGZIP(file_zeus_service_proto_rawDescData)
	})
	return file_zeus_service_proto_rawDescData
}

var file_zeus_service_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_zeus_service_proto_goTypes = []interface{}{
	(*Empty)(nil),               // 0: fort.zeus.proto.Empty
	(*Target)(nil),              // 1: fort.zeus.proto.Target
	(*StartRequest)(nil),        // 2: fort.zeus.proto.StartRequest
	(*ZoneStatus)(nil),          // 3: fort.zeus.proto.ZoneStatus
	(*Status)(nil),              // 4: fort.zeus.proto.Status
	(*timestamp.Timestamp)(nil), // 5: google.protobuf.Timestamp
}
var file_zeus_service_proto_depIdxs = []int32{
	1, // 0: fort.zeus.proto.ZoneStatus.target:type_name -> fort.zeus.proto.Target
	5, // 1: fort.zeus.proto.Status.since:type_name -> google.protobuf.Timestamp
	3, // 2: fort.zeus.proto.Status.zones:type_name -> fort.zeus.proto.ZoneStatus
	2, // 3: fort.zeus.proto.Zeus.StartClimate:input_type -> fort.zeus.proto.StartRequest
	0, // 4: fort.zeus.proto.Zeus.GetStatus:input_type -> fort.zeus.proto.Empty
	0, // 5: fort.zeus.proto.Zeus.StopClimate:input_type -> fort.zeus.proto.Empty
	0, // 6: fort.zeus.proto.Zeus.StartClimate:output_type -> fort.zeus.proto.Empty
	4, // 7: fort.zeus.proto.Zeus.GetStatus:output_type -> fort.zeus.proto.Status
	0, // 8: fort.zeus.proto.Zeus.StopClimate:output_type -> fort.zeus.proto.Empty
	6, // [6:9] is the sub-list for method output_type
	3, // [3:6] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_zeus_service_proto_init() }
func file_zeus_service_proto_init() {
	if File_zeus_service_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_zeus_service_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Empty); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_zeus_service_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Target); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_zeus_service_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StartRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_zeus_service_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ZoneStatus); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_zeus_service_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Status); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_zeus_service_proto_msgTypes[1].OneofWrappers = []interface{}{}
	file_zeus_service_proto_msgTypes[3].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_zeus_service_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_zeus_service_proto_goTypes,
		DependencyIndexes: file_zeus_service_proto_depIdxs,
		MessageInfos:      file_zeus_service_proto_msgTypes,
	}.Build()
	File_zeus_service_proto = out.File
	file_zeus_service_proto_rawDesc = nil
	file_zeus_service_proto_goTypes = nil
	file_zeus_service_proto_depIdxs = nil
}
