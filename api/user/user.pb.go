package user
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
type User struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Name          string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	CreatedAt     string                 `protobuf:"bytes,3,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	UpdatedAt     string                 `protobuf:"bytes,4,opt,name=updated_at,json=updatedAt,proto3" json:"updated_at,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *User) Reset() {
	*x = User{}
	mi := &file_proto_user_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *User) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*User) ProtoMessage() {}
func (x *User) ProtoReflect() protoreflect.Message {
	mi := &file_proto_user_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}
func (*User) Descriptor() ([]byte, []int) {
	return file_proto_user_proto_rawDescGZIP(), []int{0}
}
func (x *User) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}
func (x *User) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}
func (x *User) GetCreatedAt() string {
	if x != nil {
		return x.CreatedAt
	}
	return ""
}
func (x *User) GetUpdatedAt() string {
	if x != nil {
		return x.UpdatedAt
	}
	return ""
}
type CreateRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	TenantId      string                 `protobuf:"bytes,1,opt,name=tenant_id,json=tenantId,proto3" json:"tenant_id,omitempty"`
	Name          string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *CreateRequest) Reset() {
	*x = CreateRequest{}
	mi := &file_proto_user_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *CreateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*CreateRequest) ProtoMessage() {}
func (x *CreateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_user_proto_msgTypes[1]
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
	return file_proto_user_proto_rawDescGZIP(), []int{1}
}
func (x *CreateRequest) GetTenantId() string {
	if x != nil {
		return x.TenantId
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
	User          *User                  `protobuf:"bytes,1,opt,name=user,proto3" json:"user,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *CreateResponse) Reset() {
	*x = CreateResponse{}
	mi := &file_proto_user_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *CreateResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*CreateResponse) ProtoMessage() {}
func (x *CreateResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_user_proto_msgTypes[2]
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
	return file_proto_user_proto_rawDescGZIP(), []int{2}
}
func (x *CreateResponse) GetUser() *User {
	if x != nil {
		return x.User
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
	mi := &file_proto_user_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *GetRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*GetRequest) ProtoMessage() {}
func (x *GetRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_user_proto_msgTypes[3]
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
	return file_proto_user_proto_rawDescGZIP(), []int{3}
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
	User          *User                  `protobuf:"bytes,1,opt,name=user,proto3" json:"user,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *GetResponse) Reset() {
	*x = GetResponse{}
	mi := &file_proto_user_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *GetResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*GetResponse) ProtoMessage() {}
func (x *GetResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_user_proto_msgTypes[4]
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
	return file_proto_user_proto_rawDescGZIP(), []int{4}
}
func (x *GetResponse) GetUser() *User {
	if x != nil {
		return x.User
	}
	return nil
}
type UpdateRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	TenantId      string                 `protobuf:"bytes,1,opt,name=tenant_id,json=tenantId,proto3" json:"tenant_id,omitempty"`
	Id            string                 `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	Name          string                 `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *UpdateRequest) Reset() {
	*x = UpdateRequest{}
	mi := &file_proto_user_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *UpdateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*UpdateRequest) ProtoMessage() {}
func (x *UpdateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_user_proto_msgTypes[5]
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
	return file_proto_user_proto_rawDescGZIP(), []int{5}
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
func (x *UpdateRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}
type UpdateResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	User          *User                  `protobuf:"bytes,1,opt,name=user,proto3" json:"user,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *UpdateResponse) Reset() {
	*x = UpdateResponse{}
	mi := &file_proto_user_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *UpdateResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*UpdateResponse) ProtoMessage() {}
func (x *UpdateResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_user_proto_msgTypes[6]
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
	return file_proto_user_proto_rawDescGZIP(), []int{6}
}
func (x *UpdateResponse) GetUser() *User {
	if x != nil {
		return x.User
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
	mi := &file_proto_user_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *DeleteRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*DeleteRequest) ProtoMessage() {}
func (x *DeleteRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_user_proto_msgTypes[7]
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
	return file_proto_user_proto_rawDescGZIP(), []int{7}
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
	mi := &file_proto_user_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *DeleteResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*DeleteResponse) ProtoMessage() {}
func (x *DeleteResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_user_proto_msgTypes[8]
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
	return file_proto_user_proto_rawDescGZIP(), []int{8}
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
	mi := &file_proto_user_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *ListRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*ListRequest) ProtoMessage() {}
func (x *ListRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_user_proto_msgTypes[9]
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
	return file_proto_user_proto_rawDescGZIP(), []int{9}
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
	Users         []*User                `protobuf:"bytes,1,rep,name=users,proto3" json:"users,omitempty"`
	Total         int32                  `protobuf:"varint,2,opt,name=total,proto3" json:"total,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *ListResponse) Reset() {
	*x = ListResponse{}
	mi := &file_proto_user_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *ListResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*ListResponse) ProtoMessage() {}
func (x *ListResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_user_proto_msgTypes[10]
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
	return file_proto_user_proto_rawDescGZIP(), []int{10}
}
func (x *ListResponse) GetUsers() []*User {
	if x != nil {
		return x.Users
	}
	return nil
}
func (x *ListResponse) GetTotal() int32 {
	if x != nil {
		return x.Total
	}
	return 0
}
type SearchRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	TenantId      string                 `protobuf:"bytes,1,opt,name=tenant_id,json=tenantId,proto3" json:"tenant_id,omitempty"`
	Query         string                 `protobuf:"bytes,2,opt,name=query,proto3" json:"query,omitempty"`
	Page          int32                  `protobuf:"varint,3,opt,name=page,proto3" json:"page,omitempty"`
	PageSize      int32                  `protobuf:"varint,4,opt,name=page_size,json=pageSize,proto3" json:"page_size,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *SearchRequest) Reset() {
	*x = SearchRequest{}
	mi := &file_proto_user_proto_msgTypes[11]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *SearchRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*SearchRequest) ProtoMessage() {}
func (x *SearchRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_user_proto_msgTypes[11]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}
func (*SearchRequest) Descriptor() ([]byte, []int) {
	return file_proto_user_proto_rawDescGZIP(), []int{11}
}
func (x *SearchRequest) GetTenantId() string {
	if x != nil {
		return x.TenantId
	}
	return ""
}
func (x *SearchRequest) GetQuery() string {
	if x != nil {
		return x.Query
	}
	return ""
}
func (x *SearchRequest) GetPage() int32 {
	if x != nil {
		return x.Page
	}
	return 0
}
func (x *SearchRequest) GetPageSize() int32 {
	if x != nil {
		return x.PageSize
	}
	return 0
}
type SearchResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Users         []*User                `protobuf:"bytes,1,rep,name=users,proto3" json:"users,omitempty"`
	Total         int32                  `protobuf:"varint,2,opt,name=total,proto3" json:"total,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
func (x *SearchResponse) Reset() {
	*x = SearchResponse{}
	mi := &file_proto_user_proto_msgTypes[12]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}
func (x *SearchResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}
func (*SearchResponse) ProtoMessage() {}
func (x *SearchResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_user_proto_msgTypes[12]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}
func (*SearchResponse) Descriptor() ([]byte, []int) {
	return file_proto_user_proto_rawDescGZIP(), []int{12}
}
func (x *SearchResponse) GetUsers() []*User {
	if x != nil {
		return x.Users
	}
	return nil
}
func (x *SearchResponse) GetTotal() int32 {
	if x != nil {
		return x.Total
	}
	return 0
}
var File_proto_user_proto protoreflect.FileDescriptor
const file_proto_user_proto_rawDesc = "" +
	"\n" +
	"\x10proto/user.proto\x12\x04user\"h\n" +
	"\x04User\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12\x12\n" +
	"\x04name\x18\x02 \x01(\tR\x04name\x12\x1d\n" +
	"\n" +
	"created_at\x18\x03 \x01(\tR\tcreatedAt\x12\x1d\n" +
	"\n" +
	"updated_at\x18\x04 \x01(\tR\tupdatedAt\"@\n" +
	"\rCreateRequest\x12\x1b\n" +
	"\ttenant_id\x18\x01 \x01(\tR\btenantId\x12\x12\n" +
	"\x04name\x18\x02 \x01(\tR\x04name\"0\n" +
	"\x0eCreateResponse\x12\x1e\n" +
	"\x04user\x18\x01 \x01(\v2\n" +
	".user.UserR\x04user\"9\n" +
	"\n" +
	"GetRequest\x12\x1b\n" +
	"\ttenant_id\x18\x01 \x01(\tR\btenantId\x12\x0e\n" +
	"\x02id\x18\x02 \x01(\tR\x02id\"-\n" +
	"\vGetResponse\x12\x1e\n" +
	"\x04user\x18\x01 \x01(\v2\n" +
	".user.UserR\x04user\"P\n" +
	"\rUpdateRequest\x12\x1b\n" +
	"\ttenant_id\x18\x01 \x01(\tR\btenantId\x12\x0e\n" +
	"\x02id\x18\x02 \x01(\tR\x02id\x12\x12\n" +
	"\x04name\x18\x03 \x01(\tR\x04name\"0\n" +
	"\x0eUpdateResponse\x12\x1e\n" +
	"\x04user\x18\x01 \x01(\v2\n" +
	".user.UserR\x04user\"<\n" +
	"\rDeleteRequest\x12\x1b\n" +
	"\ttenant_id\x18\x01 \x01(\tR\btenantId\x12\x0e\n" +
	"\x02id\x18\x02 \x01(\tR\x02id\"*\n" +
	"\x0eDeleteResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess\"[\n" +
	"\vListRequest\x12\x1b\n" +
	"\ttenant_id\x18\x01 \x01(\tR\btenantId\x12\x12\n" +
	"\x04page\x18\x02 \x01(\x05R\x04page\x12\x1b\n" +
	"\tpage_size\x18\x03 \x01(\x05R\bpageSize\"F\n" +
	"\fListResponse\x12 \n" +
	"\x05users\x18\x01 \x03(\v2\n" +
	".user.UserR\x05users\x12\x14\n" +
	"\x05total\x18\x02 \x01(\x05R\x05total\"s\n" +
	"\rSearchRequest\x12\x1b\n" +
	"\ttenant_id\x18\x01 \x01(\tR\btenantId\x12\x14\n" +
	"\x05query\x18\x02 \x01(\tR\x05query\x12\x12\n" +
	"\x04page\x18\x03 \x01(\x05R\x04page\x12\x1b\n" +
	"\tpage_size\x18\x04 \x01(\x05R\bpageSize\"H\n" +
	"\x0eSearchResponse\x12 \n" +
	"\x05users\x18\x01 \x03(\v2\n" +
	".user.UserR\x05users\x12\x14\n" +
	"\x05total\x18\x02 \x01(\x05R\x05total2\xbc\x02\n" +
	"\vUserService\x123\n" +
	"\x06Create\x12\x13.user.CreateRequest\x1a\x14.user.CreateResponse\x12*\n" +
	"\x03Get\x12\x10.user.GetRequest\x1a\x11.user.GetResponse\x123\n" +
	"\x06Update\x12\x13.user.UpdateRequest\x1a\x14.user.UpdateResponse\x123\n" +
	"\x06Delete\x12\x13.user.DeleteRequest\x1a\x14.user.DeleteResponse\x12-\n" +
	"\x04List\x12\x11.user.ListRequest\x1a\x12.user.ListResponse\x123\n" +
	"\x06Search\x12\x13.user.SearchRequest\x1a\x14.user.SearchResponseB\x13Z\x11zxc/api/user;userb\x06proto3"
var (
	file_proto_user_proto_rawDescOnce sync.Once
	file_proto_user_proto_rawDescData []byte
)
func file_proto_user_proto_rawDescGZIP() []byte {
	file_proto_user_proto_rawDescOnce.Do(func() {
		file_proto_user_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proto_user_proto_rawDesc), len(file_proto_user_proto_rawDesc)))
	})
	return file_proto_user_proto_rawDescData
}
var file_proto_user_proto_msgTypes = make([]protoimpl.MessageInfo, 13)
var file_proto_user_proto_goTypes = []any{
	(*User)(nil),           
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
	(*SearchRequest)(nil),  
	(*SearchResponse)(nil), 
}
var file_proto_user_proto_depIdxs = []int32{
	0,  
	0,  
	0,  
	0,  
	0,  
	1,  
	3,  
	5,  
	7,  
	9,  
	11, 
	2,  
	4,  
	6,  
	8,  
	10, 
	12, 
	11, 
	5,  
	5,  
	5,  
	0,  
}
func init() { file_proto_user_proto_init() }
func file_proto_user_proto_init() {
	if File_proto_user_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proto_user_proto_rawDesc), len(file_proto_user_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   13,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_user_proto_goTypes,
		DependencyIndexes: file_proto_user_proto_depIdxs,
		MessageInfos:      file_proto_user_proto_msgTypes,
	}.Build()
	File_proto_user_proto = out.File
	file_proto_user_proto_goTypes = nil
	file_proto_user_proto_depIdxs = nil
}
