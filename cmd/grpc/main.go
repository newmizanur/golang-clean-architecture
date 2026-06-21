package main

import (
	"fmt"
	"net"

	"golang-clean-architecture/internal/config"
	grpcdelivery "golang-clean-architecture/internal/delivery/grpc"
	"golang-clean-architecture/internal/delivery/grpc/middleware"
	pb "golang-clean-architecture/internal/delivery/grpc/pb"
	"golang-clean-architecture/internal/repository"
	"golang-clean-architecture/internal/usecase"

	"google.golang.org/grpc"
)

func main() {
	viperConfig := config.NewViper()
	log := config.NewLogger(viperConfig)
	db := config.NewDatabase(viperConfig, log)
	validate := config.NewValidator(viperConfig)

	itemRepository := repository.NewItemRepository(db, log)
	itemUseCase := usecase.NewItemUseCase(db, log, validate, itemRepository)

	jwtSecret := viperConfig.GetString("jwt.secret")

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.UnaryAuthInterceptor(jwtSecret)),
	)

	pb.RegisterItemServiceServer(grpcServer, grpcdelivery.NewItemGRPCServer(itemUseCase))

	host := viperConfig.GetString("grpc.host")
	port := viperConfig.GetInt("grpc.port")
	addr := fmt.Sprintf("%s:%d", host, port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", addr, err)
	}

	log.Infof("gRPC server listening on %s", addr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server failed: %v", err)
	}
}
