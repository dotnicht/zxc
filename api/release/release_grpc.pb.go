package release
import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)
const _ = grpc.SupportPackageIsVersion9
const (
	ReleaseService_Create_FullMethodName = "/release.ReleaseService/Create"
	ReleaseService_Get_FullMethodName    = "/release.ReleaseService/Get"
	ReleaseService_Deploy_FullMethodName = "/release.ReleaseService/Deploy"
	ReleaseService_List_FullMethodName   = "/release.ReleaseService/List"
	ReleaseService_Search_FullMethodName = "/release.ReleaseService/Search"
)
type ReleaseServiceClient interface {
	Create(ctx context.Context, in *CreateRequest, opts ...grpc.CallOption) (*CreateResponse, error)
	Get(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (*GetResponse, error)
	Deploy(ctx context.Context, in *DeployRequest, opts ...grpc.CallOption) (*DeployResponse, error)
	List(ctx context.Context, in *ListRequest, opts ...grpc.CallOption) (*ListResponse, error)
	Search(ctx context.Context, in *SearchRequest, opts ...grpc.CallOption) (*SearchResponse, error)
}
type releaseServiceClient struct {
	cc grpc.ClientConnInterface
}
func NewReleaseServiceClient(cc grpc.ClientConnInterface) ReleaseServiceClient {
	return &releaseServiceClient{cc}
}
func (c *releaseServiceClient) Create(ctx context.Context, in *CreateRequest, opts ...grpc.CallOption) (*CreateResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(CreateResponse)
	err := c.cc.Invoke(ctx, ReleaseService_Create_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (c *releaseServiceClient) Get(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (*GetResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetResponse)
	err := c.cc.Invoke(ctx, ReleaseService_Get_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (c *releaseServiceClient) Deploy(ctx context.Context, in *DeployRequest, opts ...grpc.CallOption) (*DeployResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(DeployResponse)
	err := c.cc.Invoke(ctx, ReleaseService_Deploy_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (c *releaseServiceClient) List(ctx context.Context, in *ListRequest, opts ...grpc.CallOption) (*ListResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ListResponse)
	err := c.cc.Invoke(ctx, ReleaseService_List_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (c *releaseServiceClient) Search(ctx context.Context, in *SearchRequest, opts ...grpc.CallOption) (*SearchResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SearchResponse)
	err := c.cc.Invoke(ctx, ReleaseService_Search_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
type ReleaseServiceServer interface {
	Create(context.Context, *CreateRequest) (*CreateResponse, error)
	Get(context.Context, *GetRequest) (*GetResponse, error)
	Deploy(context.Context, *DeployRequest) (*DeployResponse, error)
	List(context.Context, *ListRequest) (*ListResponse, error)
	Search(context.Context, *SearchRequest) (*SearchResponse, error)
	mustEmbedUnimplementedReleaseServiceServer()
}
type UnimplementedReleaseServiceServer struct{}
func (UnimplementedReleaseServiceServer) Create(context.Context, *CreateRequest) (*CreateResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedReleaseServiceServer) Get(context.Context, *GetRequest) (*GetResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Get not implemented")
}
func (UnimplementedReleaseServiceServer) Deploy(context.Context, *DeployRequest) (*DeployResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Deploy not implemented")
}
func (UnimplementedReleaseServiceServer) List(context.Context, *ListRequest) (*ListResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method List not implemented")
}
func (UnimplementedReleaseServiceServer) Search(context.Context, *SearchRequest) (*SearchResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Search not implemented")
}
func (UnimplementedReleaseServiceServer) mustEmbedUnimplementedReleaseServiceServer() {}
func (UnimplementedReleaseServiceServer) testEmbeddedByValue()                        {}
type UnsafeReleaseServiceServer interface {
	mustEmbedUnimplementedReleaseServiceServer()
}
func RegisterReleaseServiceServer(s grpc.ServiceRegistrar, srv ReleaseServiceServer) {
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&ReleaseService_ServiceDesc, srv)
}
func _ReleaseService_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ReleaseServiceServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ReleaseService_Create_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ReleaseServiceServer).Create(ctx, req.(*CreateRequest))
	}
	return interceptor(ctx, in, info, handler)
}
func _ReleaseService_Get_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ReleaseServiceServer).Get(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ReleaseService_Get_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ReleaseServiceServer).Get(ctx, req.(*GetRequest))
	}
	return interceptor(ctx, in, info, handler)
}
func _ReleaseService_Deploy_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeployRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ReleaseServiceServer).Deploy(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ReleaseService_Deploy_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ReleaseServiceServer).Deploy(ctx, req.(*DeployRequest))
	}
	return interceptor(ctx, in, info, handler)
}
func _ReleaseService_List_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ReleaseServiceServer).List(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ReleaseService_List_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ReleaseServiceServer).List(ctx, req.(*ListRequest))
	}
	return interceptor(ctx, in, info, handler)
}
func _ReleaseService_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SearchRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ReleaseServiceServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ReleaseService_Search_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ReleaseServiceServer).Search(ctx, req.(*SearchRequest))
	}
	return interceptor(ctx, in, info, handler)
}
var ReleaseService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "release.ReleaseService",
	HandlerType: (*ReleaseServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Create",
			Handler:    _ReleaseService_Create_Handler,
		},
		{
			MethodName: "Get",
			Handler:    _ReleaseService_Get_Handler,
		},
		{
			MethodName: "Deploy",
			Handler:    _ReleaseService_Deploy_Handler,
		},
		{
			MethodName: "List",
			Handler:    _ReleaseService_List_Handler,
		},
		{
			MethodName: "Search",
			Handler:    _ReleaseService_Search_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/release.proto",
}
