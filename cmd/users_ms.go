package main

import (
	"net"

	"users/internal/api"
	"users/internal/services"
	cfg "users/pkg/config"
	lgr "users/pkg/logger"

	"google.golang.org/grpc"
)

func main() {
	// Init config
	_cfg, err := cfg.NewConfig("../users.cfg.toml")
	if err != nil {
		lgr.LOG.Error("_ERR_: ", "Cant init config: "+err.Error())
	}

	// Init logger
	_ = lgr.GetLogger(_cfg)
	lgr.LOG.Info("===", "Custom logger is init")

	// Init gRPC server
	lgr.LOG.Info("===", "Creating new gRPC server...")
	s := grpc.NewServer()
	srv := &services.GrpcUsersServer{}
	api.RegisterUsersServicesServer(s, srv)

	l, err := net.Listen("tcp", cfg.CFG.Server.BindAddress)
	if err != nil {
		lgr.LOG.Error("_ERR_: ", "Net listen error: "+err.Error())
	}

	lgr.LOG.Info("===", "Serving gRPC requests...")
	if err := s.Serve(l); err != nil {
		lgr.LOG.Error("_ERR_: ", "Cant serve grpc requests: "+err.Error())
	}
}
