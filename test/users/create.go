package main

import (
	"context"
	"log"

	"users/internal/api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.Dial("0.0.0.0:50066", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	request := &api.UsersRequest{
		Login:    "Vladick",
		Password: "12345",
	}

	c := api.NewUsersServicesClient(conn)
	res, err := c.CreateUser(context.Background(), request)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(res)
}
