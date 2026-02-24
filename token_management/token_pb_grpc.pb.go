package token_management

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type TokenManagerClient interface {
	WriteToken(ctx context.Context, in *WriteTokenMsg, opts ...grpc.CallOption) (*WriteResponse, error)
	ReadToken(ctx context.Context, in *Token, opts ...grpc.CallOption) (*WriteResponse, error)
	WriteBroadcast(ctx context.Context, in *WriteBroadcastRequest, opts ...grpc.CallOption) (*WriteBroadcastResponse, error)
	ReadBroadcast(ctx context.Context, in *ReadBroadcastRequest, opts ...grpc.CallOption) (*ReadBroadcastResponse, error)
}

type tokenManagerClient struct {
	cc grpc.ClientConnInterface
}

func NewTokenManagerClient(cc grpc.ClientConnInterface) TokenManagerClient {
	return &tokenManagerClient{cc}
}

func (c *tokenManagerClient) WriteToken(ctx context.Context, in *WriteTokenMsg, opts ...grpc.CallOption) (*WriteResponse, error) {
	out := new(WriteResponse)
	err := c.cc.Invoke(ctx, "/token_management.TokenManager/WriteToken", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenManagerClient) ReadToken(ctx context.Context, in *Token, opts ...grpc.CallOption) (*WriteResponse, error) {
	out := new(WriteResponse)
	err := c.cc.Invoke(ctx, "/token_management.TokenManager/ReadToken", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenManagerClient) WriteBroadcast(ctx context.Context, in *WriteBroadcastRequest, opts ...grpc.CallOption) (*WriteBroadcastResponse, error) {
	out := new(WriteBroadcastResponse)
	err := c.cc.Invoke(ctx, "/token_management.TokenManager/WriteBroadcast", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenManagerClient) ReadBroadcast(ctx context.Context, in *ReadBroadcastRequest, opts ...grpc.CallOption) (*ReadBroadcastResponse, error) {
	out := new(ReadBroadcastResponse)
	err := c.cc.Invoke(ctx, "/token_management.TokenManager/ReadBroadcast", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type TokenManagerServer interface {
	WriteToken(context.Context, *WriteTokenMsg) (*WriteResponse, error)
	ReadToken(context.Context, *Token) (*WriteResponse, error)
	WriteBroadcast(context.Context, *WriteBroadcastRequest) (*WriteBroadcastResponse, error)
	ReadBroadcast(context.Context, *ReadBroadcastRequest) (*ReadBroadcastResponse, error)
	mustEmbedUnimplementedTokenManagerServer()
}

type UnimplementedTokenManagerServer struct{}

func (UnimplementedTokenManagerServer) WriteToken(context.Context, *WriteTokenMsg) (*WriteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method WriteToken not implemented")
}

func (UnimplementedTokenManagerServer) ReadToken(context.Context, *Token) (*WriteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReadToken not implemented")
}

func (UnimplementedTokenManagerServer) WriteBroadcast(context.Context, *WriteBroadcastRequest) (*WriteBroadcastResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method WriteBroadcast not implemented")
}

func (UnimplementedTokenManagerServer) ReadBroadcast(context.Context, *ReadBroadcastRequest) (*ReadBroadcastResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReadBroadcast not implemented")
}

func (UnimplementedTokenManagerServer) mustEmbedUnimplementedTokenManagerServer() {}

func RegisterTokenManagerServer(s grpc.ServiceRegistrar, srv TokenManagerServer) {
	s.RegisterService(&TokenManager_ServiceDesc, srv)
}

func _TokenManager_WriteToken_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WriteTokenMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenManagerServer).WriteToken(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/token_management.TokenManager/WriteToken",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenManagerServer).WriteToken(ctx, req.(*WriteTokenMsg))
	}
	return interceptor(ctx, in, info, handler)
}

func _TokenManager_ReadToken_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenManagerServer).ReadToken(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/token_management.TokenManager/ReadToken",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenManagerServer).ReadToken(ctx, req.(*Token))
	}
	return interceptor(ctx, in, info, handler)
}

func _TokenManager_WriteBroadcast_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WriteBroadcastRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenManagerServer).WriteBroadcast(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/token_management.TokenManager/WriteBroadcast",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenManagerServer).WriteBroadcast(ctx, req.(*WriteBroadcastRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _TokenManager_ReadBroadcast_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReadBroadcastRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenManagerServer).ReadBroadcast(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/token_management.TokenManager/ReadBroadcast",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenManagerServer).ReadBroadcast(ctx, req.(*ReadBroadcastRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var TokenManager_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "token_management.TokenManager",
	HandlerType: (*TokenManagerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "WriteToken",
			Handler:    _TokenManager_WriteToken_Handler,
		},
		{
			MethodName: "ReadToken",
			Handler:    _TokenManager_ReadToken_Handler,
		},
		{
			MethodName: "WriteBroadcast",
			Handler:    _TokenManager_WriteBroadcast_Handler,
		},
		{
			MethodName: "ReadBroadcast",
			Handler:    _TokenManager_ReadBroadcast_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "token_management/token_pb.proto",
}
