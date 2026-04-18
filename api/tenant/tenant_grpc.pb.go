package tenant
import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)
const _ = grpc.SupportPackageIsVersion9
const (
	TenantService_Create_FullMethodName = "/tenant.TenantService/Create"
	TenantService_Get_FullMethodName    = "/tenant.TenantService/Get"
	TenantService_Update_FullMethodName = "/tenant.TenantService/Update"
	TenantService_Delete_FullMethodName = "/tenant.TenantService/Delete"
	TenantService_List_FullMethodName   = "/tenant.TenantService/List"
	TenantService_Search_FullMethodName = "/tenant.TenantService/Search"
)
type TenantServiceClient interface {
	Create(ctx context.Context, in *CreateRequest, opts ...grpc.CallOption) (*CreateResponse, error)
	Get(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (*GetResponse, error)
	Update(ctx context.Context, in *UpdateRequest, opts ...grpc.CallOption) (*UpdateResponse, error)
	Delete(ctx context.Context, in *DeleteRequest, opts ...grpc.CallOption) (*DeleteResponse, error)
	List(ctx context.Context, in *ListRequest, opts ...grpc.CallOption) (*ListResponse, error)
	Search(ctx context.Context, in *SearchRequest, opts ...grpc.CallOption) (*SearchResponse, error)
}
type tenantServiceClient struct {
	cc grpc.ClientConnInterface
}
func NewTenantServiceClient(cc grpc.ClientConnInterface) TenantServiceClient {
	return &tenantServiceClient{cc}
}
func (c *tenantServiceClient) Create(ctx context.Context, in *CreateRequest, opts ...grpc.CallOption) (*CreateResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(CreateResponse)
	err := c.cc.Invoke(ctx, TenantService_Create_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (c *tenantServiceClient) Get(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (*GetResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetResponse)
	err := c.cc.Invoke(ctx, TenantService_Get_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (c *tenantServiceClient) Update(ctx context.Context, in *UpdateRequest, opts ...grpc.CallOption) (*UpdateResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UpdateResponse)
	err := c.cc.Invoke(ctx, TenantService_Update_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (c *tenantServiceClient) Delete(ctx context.Context, in *DeleteRequest, opts ...grpc.CallOption) (*DeleteResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(DeleteResponse)
	err := c.cc.Invoke(ctx, TenantService_Delete_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (c *tenantServiceClient) List(ctx context.Context, in *ListRequest, opts ...grpc.CallOption) (*ListResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ListResponse)
	err := c.cc.Invoke(ctx, TenantService_List_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (c *tenantServiceClient) Search(ctx context.Context, in *SearchRequest, opts ...grpc.CallOption) (*SearchResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SearchResponse)
	err := c.cc.Invoke(ctx, TenantService_Search_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
type TenantServiceServer interface {
	Create(context.Context, *CreateRequest) (*CreateResponse, error)
	Get(context.Context, *GetRequest) (*GetResponse, error)
	Update(context.Context, *UpdateRequest) (*UpdateResponse, error)
	Delete(context.Context, *DeleteRequest) (*DeleteResponse, error)
	List(context.Context, *ListRequest) (*ListResponse, error)
	Search(context.Context, *SearchRequest) (*SearchResponse, error)
	mustEmbedUnimplementedTenantServiceServer()
}
type UnimplementedTenantServiceServer struct{}
func (UnimplementedTenantServiceServer) Create(context.Context, *CreateRequest) (*CreateResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedTenantServiceServer) Get(context.Context, *GetRequest) (*GetResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Get not implemented")
}
func (UnimplementedTenantServiceServer) Update(context.Context, *UpdateRequest) (*UpdateResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedTenantServiceServer) Delete(context.Context, *DeleteRequest) (*DeleteResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedTenantServiceServer) List(context.Context, *ListRequest) (*ListResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method List not implemented")
}
func (UnimplementedTenantServiceServer) Search(context.Context, *SearchRequest) (*SearchResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Search not implemented")
}
func (UnimplementedTenantServiceServer) mustEmbedUnimplementedTenantServiceServer() {}
func (UnimplementedTenantServiceServer) testEmbeddedByValue()                       {}
type UnsafeTenantServiceServer interface {
	mustEmbedUnimplementedTenantServiceServer()
}
func RegisterTenantServiceServer(s grpc.ServiceRegistrar, srv TenantServiceServer) {
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&TenantService_ServiceDesc, srv)
}
func _TenantService_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TenantServiceServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: TenantService_Create_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TenantServiceServer).Create(ctx, req.(*CreateRequest))
	}
	return interceptor(ctx, in, info, handler)
}
func _TenantService_Get_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TenantServiceServer).Get(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: TenantService_Get_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TenantServiceServer).Get(ctx, req.(*GetRequest))
	}
	return interceptor(ctx, in, info, handler)
}
func _TenantService_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TenantServiceServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: TenantService_Update_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TenantServiceServer).Update(ctx, req.(*UpdateRequest))
	}
	return interceptor(ctx, in, info, handler)
}
func _TenantService_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TenantServiceServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: TenantService_Delete_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TenantServiceServer).Delete(ctx, req.(*DeleteRequest))
	}
	return interceptor(ctx, in, info, handler)
}
func _TenantService_List_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TenantServiceServer).List(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: TenantService_List_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TenantServiceServer).List(ctx, req.(*ListRequest))
	}
	return interceptor(ctx, in, info, handler)
}
func _TenantService_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SearchRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TenantServiceServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: TenantService_Search_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TenantServiceServer).Search(ctx, req.(*SearchRequest))
	}
	return interceptor(ctx, in, info, handler)
}
var TenantService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "tenant.TenantService",
	HandlerType: (*TenantServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Create",
			Handler:    _TenantService_Create_Handler,
		},
		{
			MethodName: "Get",
			Handler:    _TenantService_Get_Handler,
		},
		{
			MethodName: "Update",
			Handler:    _TenantService_Update_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _TenantService_Delete_Handler,
		},
		{
			MethodName: "List",
			Handler:    _TenantService_List_Handler,
		},
		{
			MethodName: "Search",
			Handler:    _TenantService_Search_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/tenant.proto",
}
