package utils

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/container-storage-interface/spec/lib/go/csi/v0"
)

// ChainUnaryClient chains one or more unary, client interceptors
// together into a left-to-right series that can be provided to a
// new gRPC client.
func ChainUnaryClient(
	i ...grpc.UnaryClientInterceptor) grpc.UnaryClientInterceptor {

	switch len(i) {
	case 0:
		return func(
			ctx context.Context,
			method string,
			req, rep interface{},
			cc *grpc.ClientConn,
			invoker grpc.UnaryInvoker,
			opts ...grpc.CallOption) error {
			return invoker(ctx, method, req, rep, cc, opts...)
		}
	case 1:
		return i[0]
	}

	return func(
		ctx context.Context,
		method string,
		req, rep interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption) error {

		bc := func(
			cur grpc.UnaryClientInterceptor,
			nxt grpc.UnaryInvoker) grpc.UnaryInvoker {

			return func(
				curCtx context.Context,
				curMethod string,
				curReq, curRep interface{},
				curCC *grpc.ClientConn,
				curOpts ...grpc.CallOption) error {

				return cur(
					curCtx,
					curMethod,
					curReq, curRep,
					curCC, nxt,
					curOpts...)
			}
		}

		c := invoker
		for j := len(i) - 1; j >= 0; j-- {
			c = bc(i[j], c)
		}

		return c(ctx, method, req, rep, cc, opts...)
	}
}

// ChainUnaryServer chains one or more unary, server interceptors
// together into a left-to-right series that can be provided to a
// new gRPC server.
func ChainUnaryServer(
	i ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {

	switch len(i) {
	case 0:
		return func(
			ctx context.Context,
			req interface{},
			_ *grpc.UnaryServerInfo,
			handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}
	case 1:
		return i[0]
	}

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {

		bc := func(
			cur grpc.UnaryServerInterceptor,
			nxt grpc.UnaryHandler) grpc.UnaryHandler {
			return func(
				curCtx context.Context,
				curReq interface{}) (interface{}, error) {
				return cur(curCtx, curReq, info, nxt)
			}
		}
		c := handler
		for j := len(i) - 1; j >= 0; j-- {
			c = bc(i[j], c)
		}
		return c(ctx, req)
	}
}

// nilResponses exceeds the 80char code limit, but to modify it would render
// it less readable than leaving it as is
var nilResponses = map[string]interface{}{
	CreateVolume:               (*csi.CreateVolumeResponse)(nil),
	DeleteVolume:               (*csi.DeleteVolumeResponse)(nil),
	ControllerPublishVolume:    (*csi.ControllerPublishVolumeResponse)(nil),
	ControllerUnpublishVolume:  (*csi.ControllerUnpublishVolumeResponse)(nil),
	ValidateVolumeCapabilities: (*csi.ValidateVolumeCapabilitiesResponse)(nil),
	ListVolumes:                (*csi.ListVolumesResponse)(nil),
	GetCapacity:                (*csi.GetCapacityResponse)(nil),
	ControllerGetCapabilities:  (*csi.ControllerGetCapabilitiesResponse)(nil),
	GetPluginInfo:              (*csi.GetPluginInfoResponse)(nil),
	NodeGetId:                  (*csi.NodeGetIdResponse)(nil),
	NodePublishVolume:          (*csi.NodePublishVolumeResponse)(nil),
	NodeUnpublishVolume:        (*csi.NodeUnpublishVolumeResponse)(nil),
	NodeGetCapabilities:        (*csi.NodeGetCapabilitiesResponse)(nil),
}

// IsNilResponse returns a flag indicating whether or not the provided
// response object is a nil object wrapped inside a non-nil interface.
func IsNilResponse(method string, rep interface{}) bool {
	// Determine whether or not the resposne is nil. Otherwise it
	// will no longer be possible to perform a nil equality check on the
	// response to the interface{} rules for nil comparison. For more info
	// please see https://golang.org/doc/faq#nil_error and
	// https://github.com/grpc/grpc-go/issues/532.
	if rep == nil {
		return true
	}
	if nilRep := nilResponses[method]; rep == nilRep {
		return true
	}
	return false
}
