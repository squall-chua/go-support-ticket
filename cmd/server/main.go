package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
	"github.com/squall-chua/go-support-ticket/internal/repository"
	"github.com/squall-chua/go-support-ticket/internal/service"
	"github.com/squall-chua/go-support-ticket/pkg/event"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	pgURI := os.Getenv("PG_URI")
	if pgURI == "" {
		pgURI = "postgres://postgres:postgres@localhost:5432/support_ticket"
	}

	connector, err := repository.NewConnector(ctx, repository.Config{
		PostgresDSN: pgURI,
		MongoURI:    mongoURI,
		MongoDBName: "support_ticket",
	})
	if err != nil {
		log.Fatalf("Failed to initialize database connector: %v", err)
	}
	defer connector.Close()

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	publisher, err := event.NewNatsPublisher(nc)
	if err != nil {
		log.Fatalf("Failed to create NATS publisher: %v", err)
	}

	repos := repository.NewRepositories(connector)

	ticketSvc := service.NewTicketServiceServer(repos.Tickets, publisher)
	actionSvc := service.NewActionServiceServer(repos.Executions, repos.ActionSchemas, repos.Approvals, publisher)
	auditSvc := service.NewAuditServiceServer(repos.Audit, publisher)
	approvalSvc := service.NewApprovalServiceServer(repos.Approvals, publisher)

	grpcServer := grpc.NewServer()
	apiv1.RegisterTicketServiceServer(grpcServer, ticketSvc)
	apiv1.RegisterActionServiceServer(grpcServer, actionSvc)
	apiv1.RegisterAuditServiceServer(grpcServer, auditSvc)
	apiv1.RegisterApprovalServiceServer(grpcServer, approvalSvc)
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	grpcAddr := ":8081"

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcAddr, err)
	}

	go func() {
		log.Printf("Starting gRPC server on port %s", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	if err := apiv1.RegisterTicketServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}
	if err := apiv1.RegisterActionServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}
	if err := apiv1.RegisterAuditServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}
	if err := apiv1.RegisterApprovalServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}

	httpMux := http.NewServeMux()
	httpMux.Handle("/", mux)
	httpMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	srv := &http.Server{
		Addr: fmt.Sprintf(":%s", port),
		Handler:           httpMux,
		ReadHeaderTimeout: time.Second * 10,
	}

	go func() {
		log.Printf("Starting HTTP API Gateway server on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctxShutDown, cancelShutDown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutDown()

	grpcServer.GracefulStop()
	if err := srv.Shutdown(ctxShutDown); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
