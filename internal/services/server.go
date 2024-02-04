package services

import (
	"users/internal/api"
)

type GrpcUsersServer struct {
	api.UsersServicesServer
}
