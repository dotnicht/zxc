package payload
import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)
const (
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)
type Payload struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Path          string                 `protobuf:"bytes,2,opt,name=path,proto3" json:"path,omitempty"`
	OwnerId       string                 `protobuf:"bytes,3,opt,name=owner_id,json=ownerId,proto3" json:"owner_id,omitempty"`
	CreatedAt     string                 `protobuf:"bytes,4,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	UpdatedAt     string                 `protobuf:"bytes,5,opt,name=updated_at,json=updatedAt,proto3" json:"updated_at,omitempty"`
	Start         string                 `protobuf:"bytes,6,opt,name=start,proto3" json:"start,omitempty"`
	Stop          string                 `protobuf:"bytes,7,opt,name=stop,proto3" json:"stop,omitempty"`
	Config        string                 `protobuf:"bytes,8,opt,name=config,proto3" json:"config,omitempty"`
	Name          string                 `protobuf:"bytes,9,opt,name=name,proto3" json:"name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *Payload) Reset() {
	*x = Payload{}
	mi := &file_proto_payload_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *Payload) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*Payload) ProtoMessage() {}
func (x *Payload) ProtoReflect() protoreflect.Message {
	mi := &file_proto_payload_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}
func (*Payload) Descriptor() ([]byte, []int) {
	return file_proto_payload_proto_rawDescGZIP(), []int{0}
}
func (x *Payload) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}
func (x *Payload) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}
func (x *Payload) GetOwnerId() string {
	if x != nil {
		return x.OwnerId
	}
	return ""
}
func (x *Payload) GetCreatedAt() string {
	if x != nil {
		return x.CreatedAt
	}
	return ""
}
func (x *Payload) GetUpdatedAt() string {
	if x != nil {
		return x.UpdatedAt
	}
	return ""
}
func (x *Payload) GetStart() string {
	if x != nil {
		return x.Start
	}
	return ""
}
func (x *Payload) GetStop() string {
	if x != nil {
		return x.Stop
	}
	return ""
}
func (x *Payload) GetConfig() string {
	if x != nil {
		return x.Config
	}
	return ""
}
func (x *Payload) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}
type CreateRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	TenantId      string                 `protobuf:"bytes,1,opt,name=tenant_id,json=tenantId,proto3" json:"tenant_id,omitempty"`
	OwnerId       string                 `protobuf:"bytes,2,opt,name=owner_id,json=ownerId,proto3" json:"owner_id,omitempty"`
	Content       []byte                 `protobuf:"bytes,3,opt,name=content,proto3" json:"content,omitempty"`
	Start         string                 `protobuf:"bytes,4,opt,name=start,proto3" json:"start,omitempty"`
	Stop          string                 `protobuf:"bytes,5,opt,name=stop,proto3" json:"stop,omitempty"`
	Config        string                 `protobuf:"bytes,6,opt,name=config,proto3" json:"config,omitempty"`
	Name          string                 `protobuf:"bytes,7,opt,name=name,proto3" json:"name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *CreateRequest) Reset() {
	*x = CreateRequest{}
	mi := &file_proto_payload_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *CreateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*CreateRequest) ProtoMessage() {}
func (x *CreateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_payload_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}
func (*CreateRequest) Descriptor() ([]byte, []int) {
	return file_proto_payload_proto_rawDescGZIP(), []int{1}
}
func (x *CreateRequest) GetTenantId() string {
	if x != nil {
		return x.TenantId
	}
	return ""
}
func (x *CreateRequest) GetOwnerId() string {
	if x != nil {
		return x.OwnerId
	}
	return ""
}
func (x *CreateRequest) GetContent() []byte {
	if x != nil {
		return x.Content
	}
	return nil
}
func (x *CreateRequest) GetStart() string {
	if x != nil {
		return x.Start
	}
	return ""
}
func (x *CreateRequest) GetStop() string {
	if x != nil {
		return x.Stop
	}
	return ""
}
func (x *CreateRequest) GetConfig() string {
	if x != nil {
		return x.Config
	}
	return ""
}
func (x *CreateRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}
type CreateResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Payload       *Payload               `protobuf:"bytes,1,opt,name=payload,proto3" json:"payload,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *CreateResponse) Reset() {
	*x = CreateResponse{}
	mi := &file_proto_payload_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *CreateResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*CreateResponse) ProtoMessage() {}
func (x *CreateResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_payload_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}
func (*CreateResponse) Descriptor() ([]byte, []int) {
	return file_proto_payload_proto_rawDescGZIP(), []int{2}
}
func (x *CreateResponse) GetPayload() *Payload {
	if x != nil {
		return x.Payload
	}
	return nil
}
type GetRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	TenantId      string                 `protobuf:"bytes,1,opt,name=tenant_id,json=tenantId,proto3" json:"tenant_id,omitempty"`
	Id            string                 `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *GetRequest) Reset() {
	*x = GetRequest{}
	mi := &file_proto_payload_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *GetRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*GetRequest) ProtoMessage() {}
func (x *GetRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_payload_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}
func (*GetRequest) Descriptor() ([]byte, []int) {
	return file_proto_payload_proto_rawDescGZIP(), []int{3}
}
func (x *GetRequest) GetTenantId() string {
	if x != nil {
		return x.TenantId
	}
	return ""
}
func (x *GetRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}
type GetResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Payload       *Payload               `protobuf:"bytes,1,opt,name=payload,proto3" json:"payload,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *GetResponse) Reset() {
	*x = GetResponse{}
	mi := &file_proto_payload_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *GetResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*GetResponse) ProtoMessage() {}
func (x *GetResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_payload_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}
func (*GetResponse) Descriptor() ([]byte, []int) {
	return file_proto_payload_proto_rawDescGZIP(), []int{4}
}
func (x *GetResponse) GetPayload() *Payload {
	if x != nil {
		return x.Payload
	}
	return nil
}
type UpdateRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	TenantId      string                 `protobuf:"bytes,1,opt,name=tenant_id,json=tenantId,proto3" json:"tenant_id,omitempty"`
	Id            string                 `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	Path          string                 `protobuf:"bytes,3,opt,name=path,proto3" json:"path,omitempty"`
	Start         string                 `protobuf:"bytes,4,opt,name=start,proto3" json:"start,omitempty"`
	Stop          string                 `protobuf:"bytes,5,opt,name=stop,proto3" json:"stop,omitempty"`
	Config        string                 `protobuf:"bytes,6,opt,name=config,proto3" json:"config,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *UpdateRequest) Reset() {
	*x = UpdateRequest{}
	mi := &file_proto_payload_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *UpdateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*UpdateRequest) ProtoMessage() {}
func (x *UpdateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_payload_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}
func (*UpdateRequest) Descriptor() ([]byte, []int) {
	return file_proto_payload_proto_rawDescGZIP(), []int{5}
}
func (x *UpdateRequest) GetTenantId() string {
	if x != nil {
		return x.TenantId
	}
	return ""
}
func (x *UpdateRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}
func (x *UpdateRequest) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}
func (x *UpdateRequest) GetStart() string {
	if x != nil {
		return x.Start
	}
	return ""
}
func (x *UpdateRequest) GetStop() string {
	if x != nil {
		return x.Stop
	}
	return ""
}
func (x *UpdateRequest) GetConfig() string {
	if x != nil {
		return x.Config
	}
	return ""
}
type UpdateResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Payload       *Payload               `protobuf:"bytes,1,opt,name=payload,proto3" json:"payload,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *UpdateResponse) Reset() {
	*x = UpdateResponse{}
	mi := &file_proto_payload_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *UpdateResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*UpdateResponse) ProtoMessage() {}
func (x *UpdateResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_payload_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}
func (*UpdateResponse) Descriptor() ([]byte, []int) {
	return file_proto_payload_proto_rawDescGZIP(), []int{6}
}
func (x *UpdateResponse) GetPayload() *Payload {
	if x != nil {
		return x.Payload
	}
	return nil
}
type DeleteRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	TenantId      string                 `protobuf:"bytes,1,opt,name=tenant_id,json=tenantId,proto3" json:"tenant_id,omitempty"`
	Id            string                 `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *DeleteRequest) Reset() {
	*x = DeleteRequest{}
	mi := &file_proto_payload_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *DeleteRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*DeleteRequest) ProtoMessage() {}
func (x *DeleteRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_payload_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}
func (*DeleteRequest) Descriptor() ([]byte, []int) {
	return file_proto_payload_proto_rawDescGZIP(), []int{7}
}
func (x *DeleteRequest) GetTenantId() string {
	if x != nil {
		return x.TenantId
	}
	return ""
}
func (x *DeleteRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}
type DeleteResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *DeleteResponse) Reset() {
	*x = DeleteResponse{}
	mi := &file_proto_payload_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *DeleteResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*DeleteResponse) ProtoMessage() {}
func (x *DeleteResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_payload_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}
func (*DeleteResponse) Descriptor() ([]byte, []int) {
	return file_proto_payload_proto_rawDescGZIP(), []int{8}
}
func (x *DeleteResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}
type ListRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	TenantId      string                 `protobuf:"bytes,1,opt,name=tenant_id,json=tenantId,proto3" json:"tenant_id,omitempty"`
	Page          int32                  `protobuf:"varint,2,opt,name=page,proto3" json:"page,omitempty"`
	PageSize      int32                  `protobuf:"varint,3,opt,name=page_size,json=pageSize,proto3" json:"page_size,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *ListRequest) Reset() {
	*x = ListRequest{}
	mi := &file_proto_payload_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *ListRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*ListRequest) ProtoMessage() {}
func (x *ListRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_payload_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}
func (*ListRequest) Descriptor() ([]byte, []int) {
	return file_proto_payload_proto_rawDescGZIP(), []int{9}
}
func (x *ListRequest) GetTenantId() string {
	if x != nil {
		return x.TenantId
	}
	return ""
}
func (x *ListRequest) GetPage() int32 {
	if x != nil {
		return x.Page
	}
	return 0
}
func (x *ListRequest) GetPageSize() int32 {
	if x != nil {
		return x.PageSize
	}
	return 0
}
type ListResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Payloads      []*Payload             `protobuf:"bytes,1,rep,name=payloads,proto3" json:"payloads,omitempty"`
	Total         int32                  `protobuf:"varint,2,opt,name=total,proto3" json:"total,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *ListResponse) Reset() {
	*x = ListResponse{}
	mi := &file_proto_payload_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *ListResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*ListResponse) ProtoMessage() {}
func (x *ListResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_payload_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}
func (*ListResponse) Descriptor() ([]byte, []int) {
	return file_proto_payload_proto_rawDescGZIP(), []int{10}
}
func (x *ListResponse) GetPayloads() []*Payload {
	if x != nil {
		return x.Payloads
	}
	return nil
}
func (x *ListResponse) GetTotal() int32 {
	if x != nil {
		return x.Total
	}
	return 0
}
var File_proto_payload_proto protoreflect.FileDescriptor
const file_proto_payload_proto_rawDesc = "" +
	"\n" +
	"\x13proto/payload.proto\x12\apayload\"\xdc\x01\n" +
	"\aPayload\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12\x12\n" +
	"\x04path\x18\x02 \x01(\tR\x04path\x12\x19\n" +
	"\bowner_id\x18\x03 \x01(\tR\aownerId\x12\x1d\n" +
	"\n" +
	"created_at\x18\x04 \x01(\tR\tcreatedAt\x12\x1d\n" +
	"\n" +
	"updated_at\x18\x05 \x01(\tR\tupdatedAt\x12\x14\n" +
	"\x05start\x18\x06 \x01(\tR\x05start\x12\x12\n" +
	"\x04stop\x18\a \x01(\tR\x04stop\x12\x16\n" +
	"\x06config\x18\b \x01(\tR\x06config\x12\x12\n" +
	"\x04name\x18\t \x01(\tR\x04name\"\xb7\x01\n" +
	"\rCreateRequest\x12\x1b\n" +
	"\ttenant_id\x18\x01 \x01(\tR\btenantId\x12\x19\n" +
	"\bowner_id\x18\x02 \x01(\tR\aownerId\x12\x18\n" +
	"\acontent\x18\x03 \x01(\fR\acontent\x12\x14\n" +
	"\x05start\x18\x04 \x01(\tR\x05start\x12\x12\n" +
	"\x04stop\x18\x05 \x01(\tR\x04stop\x12\x16\n" +
	"\x06config\x18\x06 \x01(\tR\x06config\x12\x12\n" +
	"\x04name\x18\a \x01(\tR\x04name\"<\n" +
	"\x0eCreateResponse\x12*\n" +
	"\apayload\x18\x01 \x01(\v2\x10.payload.PayloadR\apayload\"9\n" +
	"\n" +
	"GetRequest\x12\x1b\n" +
	"\ttenant_id\x18\x01 \x01(\tR\btenantId\x12\x0e\n" +
	"\x02id\x18\x02 \x01(\tR\x02id\"9\n" +
	"\vGetResponse\x12*\n" +
	"\apayload\x18\x01 \x01(\v2\x10.payload.PayloadR\apayload\"\x92\x01\n" +
	"\rUpdateRequest\x12\x1b\n" +
	"\ttenant_id\x18\x01 \x01(\tR\btenantId\x12\x0e\n" +
	"\x02id\x18\x02 \x01(\tR\x02id\x12\x12\n" +
	"\x04path\x18\x03 \x01(\tR\x04path\x12\x14\n" +
	"\x05start\x18\x04 \x01(\tR\x05start\x12\x12\n" +
	"\x04stop\x18\x05 \x01(\tR\x04stop\x12\x16\n" +
	"\x06config\x18\x06 \x01(\tR\x06config\"<\n" +
	"\x0eUpdateResponse\x12*\n" +
	"\apayload\x18\x01 \x01(\v2\x10.payload.PayloadR\apayload\"<\n" +
	"\rDeleteRequest\x12\x1b\n" +
	"\ttenant_id\x18\x01 \x01(\tR\btenantId\x12\x0e\n" +
	"\x02id\x18\x02 \x01(\tR\x02id\"*\n" +
	"\x0eDeleteResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess\"[\n" +
	"\vListRequest\x12\x1b\n" +
	"\ttenant_id\x18\x01 \x01(\tR\btenantId\x12\x12\n" +
	"\x04page\x18\x02 \x01(\x05R\x04page\x12\x1b\n" +
	"\tpage_size\x18\x03 \x01(\x05R\bpageSize\"R\n" +
	"\fListResponse\x12,\n" +
	"\bpayloads\x18\x01 \x03(\v2\x10.payload.PayloadR\bpayloads\x12\x14\n" +
	"\x05total\x18\x02 \x01(\x05R\x05total2\xa8\x02\n" +
	"\x0ePayloadService\x129\n" +
	"\x06Create\x12\x16.payload.CreateRequest\x1a\x17.payload.CreateResponse\x120\n" +
	"\x03Get\x12\x13.payload.GetRequest\x1a\x14.payload.GetResponse\x129\n" +
	"\x06Update\x12\x16.payload.UpdateRequest\x1a\x17.payload.UpdateResponse\x129\n" +
	"\x06Delete\x12\x16.payload.DeleteRequest\x1a\x17.payload.DeleteResponse\x123\n" +
	"\x04List\x12\x14.payload.ListRequest\x1a\x15.payload.ListResponseB\x19Z\x17zxc/api/payload;payloadb\x06proto3"
var (
	file_proto_payload_proto_rawDescOnce sync.Once
	file_proto_payload_proto_rawDescData []byte
)
func file_proto_payload_proto_rawDescGZIP() []byte {
	file_proto_payload_proto_rawDescOnce.Do(func() {
		file_proto_payload_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proto_payload_proto_rawDesc), len(file_proto_payload_proto_rawDesc)))
	})
	return file_proto_payload_proto_rawDescData
}
var file_proto_payload_proto_msgTypes = make([]protoimpl.MessageInfo, 11)
var file_proto_payload_proto_goTypes = []any{
	(*Payload)(nil),        
	(*CreateRequest)(nil),  
	(*CreateResponse)(nil), 
	(*GetRequest)(nil),     
	(*GetResponse)(nil),    
	(*UpdateRequest)(nil),  
	(*UpdateResponse)(nil), 
	(*DeleteRequest)(nil),  
	(*DeleteResponse)(nil), 
	(*ListRequest)(nil),    
	(*ListResponse)(nil),   
}
var file_proto_payload_proto_depIdxs = []int32{
	0,  
	0,  
	0,  
	0,  
	1,  
	3,  
	5,  
	7,  
	9,  
	2,  
	4,  
	6,  
	8,  
	10, 
	9,  
	4,  
	4,  
	4,  
	0,  
}
func init() { file_proto_payload_proto_init() }
func file_proto_payload_proto_init() {
	if File_proto_payload_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proto_payload_proto_rawDesc), len(file_proto_payload_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   11,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_payload_proto_goTypes,
		DependencyIndexes: file_proto_payload_proto_depIdxs,
		MessageInfos:      file_proto_payload_proto_msgTypes,
	}.Build()
	File_proto_payload_proto = out.File
	file_proto_payload_proto_goTypes = nil
	file_proto_payload_proto_depIdxs = nil
}
