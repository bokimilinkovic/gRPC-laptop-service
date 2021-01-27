package service

import (
	"context"

	"gitlab.techschool.pcbook/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthServer struct {
	userStore  UserStore
	jwtManager *JWTManager
}

func NewAuthServer(userStore UserStore, jwtManager *JWTManager) *AuthServer {
	return &AuthServer{userStore, jwtManager}
}

func (as *AuthServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, err := as.userStore.Find(req.GetUsername())
	if err != nil {
		return nil, err
	}

	if user == nil || !user.IsCorrectPassword(req.GetPassword()) {
		return nil, status.Errorf(codes.NotFound, "incorect username/password")
	}

	token, err := as.jwtManager.Generate(user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot generate tkoen %v", err)
	}

	res := &pb.LoginResponse{
		AccessToken: token,
	}
	return res, nil
}
