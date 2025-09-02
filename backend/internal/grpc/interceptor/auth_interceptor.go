package interceptor

import (
	"context"
	"strings"

	"github.com/rudraprasaaad/task-scheduler/internal/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const GrpcUserContextKey = contextKey("user_claims")

type AuthInterceptor struct {
	jwtSecret string
}

func NewAuthInterceptor(jwtSecret string) *AuthInterceptor {
	return &AuthInterceptor{jwtSecret: jwtSecret}
}

func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		newCtx, err := i.authenticate(ctx)
		if err != nil {
			return nil, err
		}
		return handler(newCtx, req)
	}
}

func (i *AuthInterceptor) authenticate(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return nil, status.Error(codes.Unauthenticated, "Authorization token is not provided")
	}

	token := strings.TrimPrefix(authHeaders[0], "Bearer ")
	claims, err := auth.ValidateToken(token, i.jwtSecret)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	return context.WithValue(ctx, GrpcUserContextKey, claims), nil
}
