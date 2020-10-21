// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package topologyv1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion7

// TopologyAPIClient is the client API for TopologyAPI service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type TopologyAPIClient interface {
	GetTopology(ctx context.Context, in *GetTopologyRequest, opts ...grpc.CallOption) (*GetTopologyResponse, error)
}

type topologyAPIClient struct {
	cc grpc.ClientConnInterface
}

func NewTopologyAPIClient(cc grpc.ClientConnInterface) TopologyAPIClient {
	return &topologyAPIClient{cc}
}

func (c *topologyAPIClient) GetTopology(ctx context.Context, in *GetTopologyRequest, opts ...grpc.CallOption) (*GetTopologyResponse, error) {
	out := new(GetTopologyResponse)
	err := c.cc.Invoke(ctx, "/clutch.topology.v1.TopologyAPI/GetTopology", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// TopologyAPIServer is the server API for TopologyAPI service.
// All implementations should embed UnimplementedTopologyAPIServer
// for forward compatibility
type TopologyAPIServer interface {
	GetTopology(context.Context, *GetTopologyRequest) (*GetTopologyResponse, error)
}

// UnimplementedTopologyAPIServer should be embedded to have forward compatible implementations.
type UnimplementedTopologyAPIServer struct {
}

func (UnimplementedTopologyAPIServer) GetTopology(context.Context, *GetTopologyRequest) (*GetTopologyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTopology not implemented")
}

// UnsafeTopologyAPIServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to TopologyAPIServer will
// result in compilation errors.
type UnsafeTopologyAPIServer interface {
	mustEmbedUnimplementedTopologyAPIServer()
}

func RegisterTopologyAPIServer(s *grpc.Server, srv TopologyAPIServer) {
	s.RegisterService(&_TopologyAPI_serviceDesc, srv)
}

func _TopologyAPI_GetTopology_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetTopologyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TopologyAPIServer).GetTopology(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/clutch.topology.v1.TopologyAPI/GetTopology",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TopologyAPIServer).GetTopology(ctx, req.(*GetTopologyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _TopologyAPI_serviceDesc = grpc.ServiceDesc{
	ServiceName: "clutch.topology.v1.TopologyAPI",
	HandlerType: (*TopologyAPIServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetTopology",
			Handler:    _TopologyAPI_GetTopology_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "topology/v1/topology_api.proto",
}
