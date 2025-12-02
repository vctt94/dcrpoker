package server

import (
	"context"
	"strings"

	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// authUnaryInterceptor returns a gRPC unary interceptor that enforces
// token-based authentication for selected public RPC methods.
//
// Currently it protects the following LobbyService methods:
//   - CreateTable
//   - JoinTable
//   - SetPlayerReady
//   - SetPlayerUnready
//   - LeaveTable
//
// Only NewServer wires this middleware into the production gRPC server; direct
// method calls in unit tests and NewTestServer instances are unaffected.
func authUnaryInterceptor(s *Server) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !methodRequiresToken(info.FullMethod) {
			return handler(ctx, req)
		}

		token := extractTokenFromMetadata(ctx)
		if token == "" {
			return nil, status.Error(codes.Unauthenticated, "token required")
		}

		sess, ok := s.sessionForToken(token)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired session")
		}

		// Enforce that the bound session user matches the request's PlayerId
		// for methods that operate on a player.
		if err := enforcePlayerBinding(sess, info.FullMethod, req); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

// methodRequiresToken returns true for RPC methods that require an auth token
// when invoked over gRPC. Update this switch alongside authUnaryInterceptor's
// documentation if additional public methods become auth-protected.
func methodRequiresToken(fullMethod string) bool {
	switch fullMethod {
	case pokerrpc.LobbyService_CreateTable_FullMethodName,
		pokerrpc.LobbyService_JoinTable_FullMethodName,
		pokerrpc.LobbyService_SetPlayerReady_FullMethodName,
		pokerrpc.LobbyService_SetPlayerUnready_FullMethodName,
		pokerrpc.LobbyService_LeaveTable_FullMethodName:
		return true
	default:
		return false
	}
}

// extractTokenFromMetadata extracts the "token" value from incoming gRPC
// metadata, trimming whitespace.
func extractTokenFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	vals := md.Get("token")
	if len(vals) == 0 {
		return ""
	}
	return strings.TrimSpace(vals[0])
}

// enforcePlayerBinding verifies that the authenticated session user matches
// the PlayerId present in the request for RPCs that act on a player.
func enforcePlayerBinding(sess sessionInfo, fullMethod string, req any) error {
	switch fullMethod {
	case pokerrpc.LobbyService_CreateTable_FullMethodName:
		if r, ok := req.(*pokerrpc.CreateTableRequest); ok {
			if strings.TrimSpace(r.PlayerId) == "" {
				return status.Error(codes.InvalidArgument, "player_id required")
			}
			if sess.userID.String() != strings.TrimSpace(r.PlayerId) {
				return status.Error(codes.PermissionDenied, "token user ID does not match requested player ID")
			}
		}
	case pokerrpc.LobbyService_JoinTable_FullMethodName:
		if r, ok := req.(*pokerrpc.JoinTableRequest); ok {
			if strings.TrimSpace(r.PlayerId) == "" {
				return status.Error(codes.InvalidArgument, "player_id required")
			}
			if sess.userID.String() != strings.TrimSpace(r.PlayerId) {
				return status.Error(codes.PermissionDenied, "token user ID does not match requested player ID")
			}
		}
	case pokerrpc.LobbyService_SetPlayerReady_FullMethodName,
		pokerrpc.LobbyService_SetPlayerUnready_FullMethodName,
		pokerrpc.LobbyService_LeaveTable_FullMethodName:
		// For these calls we only enforce that the session belongs to the
		// declared PlayerId when a PlayerId is provided.
		type hasPlayerID interface {
			GetPlayerId() string
		}
		if r, ok := req.(hasPlayerID); ok {
			playerID := strings.TrimSpace(r.GetPlayerId())
			if playerID != "" && sess.userID.String() != playerID {
				return status.Error(codes.PermissionDenied, "token user ID does not match requested player ID")
			}
		}
	}

	return nil
}
