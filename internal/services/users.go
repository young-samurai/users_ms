package services

import (
	"context"
	"users/internal/api"

	stg "users/internal/storage/users"
	lgr "users/pkg/logger"
)

func (s *GrpcUsersServer) CreateUser(ctx context.Context, req *api.UsersRequest) (*api.UsersResponse, error) {
	lgr.LOG.Info("-->> ", "services.CreateUser()")

	users := stg.CreateUser(ctx, req)

	lgr.LOG.Info("-->> ", "services.CreateUser()")
	return &api.UsersResponse{Users: users}, nil
}
