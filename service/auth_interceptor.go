package service

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthInterceptop struct {
	jwtManager      *JWTManager
	accessibleRoles map[string][]string
}

func NewAuthInterceptor(jwtManager *JWTManager, accessibleRoles map[string][]string) *AuthInterceptop {
	return &AuthInterceptop{jwtManager, accessibleRoles}
}

func (interceptor *AuthInterceptop) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		log.Println("--> unary interceptor", info.FullMethod)

		err := interceptor.authorize(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func (interceptor *AuthInterceptop) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		log.Println("--> stream intercepter", info.FullMethod)

		err := interceptor.authorize(ss.Context(), info.FullMethod)
		if err != nil {
			return err
		}

		return handler(srv, ss)
	}

}

func (interceptor *AuthInterceptop) authorize(ctx context.Context, method string) error {
	accesibleRoles, ok := interceptor.accessibleRoles[method]
	if !ok {
		// everyone can access
		return nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}

	values := md["authorization"]
	if len(values) == 0 {
		return status.Errorf(codes.Unauthenticated, "authorization token is not provided")
	}

	accessToken := values[0]
	claims, err := interceptor.jwtManager.Verify(accessToken)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "access token not valid")
	}

	for _, role := range accesibleRoles {
		if role == claims.Role {
			return nil
		}
	}

	return status.Error(codes.PermissionDenied, "no permission for access this RPC")
}
