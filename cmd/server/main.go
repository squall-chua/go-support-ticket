package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/squall-chua/go-event-pubsub/pkg/broker/kafka"
	"github.com/squall-chua/go-event-pubsub/pkg/event"
	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
	"github.com/squall-chua/go-support-ticket/internal/middleware"
	"github.com/squall-chua/go-support-ticket/internal/repository"
	"github.com/squall-chua/go-support-ticket/internal/service"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		mongoURI    string
		kafkaURL    string
		port        string
		corsOrigins string
		jwtSecret   string
	)

	flag.StringVar(&mongoURI, "mongo-uri", getEnv("MONGO_URI", "mongodb://localhost:27017/support_ticket"), "MongoDB connection URI")
	flag.StringVar(&kafkaURL, "kafka-url", getEnv("KAFKA_URL", "localhost:9092"), "Kafka URL")
	flag.StringVar(&port, "port", getEnv("PORT", "8080"), "Port to listen on for both gRPC and HTTP")
	flag.StringVar(&corsOrigins, "cors-origins", getEnv("CORS_ORIGINS", "*"), "Comma-separated list of allowed CORS origins")
	flag.StringVar(&jwtSecret, "jwt-secret", getEnv("JWT_SECRET", "secret"), "JWT signing secret")
	var schemaRegistryPath string
	flag.StringVar(&schemaRegistryPath, "schema-config", getEnv("SCHEMA_CONFIG", "configs/schemas.yaml"), "Path to the event schema configuration file")

	flag.Parse()

	if mongoURI == "" {
		log.Fatalf("MongoDB URI is empty")
	}

	cs, err := connstring.ParseAndValidate(mongoURI)
	if err != nil {
		log.Fatalf("Failed to parse mongo uri: %v", err)
	}
	dbName := cs.Database
	if dbName == "" {
		log.Fatalf("Mongo database name is not specified in URI")
	}

	clientOpt := options.Client().ApplyURI(mongoURI)
	mongoC, err := mongo.Connect(clientOpt)
	if err != nil {
		log.Fatalf("Failed to connect to mongo: %v", err)
	}
	defer func() {
		if err := mongoC.Disconnect(context.Background()); err != nil {
			log.Printf("Failed to disconnect mongo: %v", err)
		}
	}()

	if err := mongoC.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping mongo: %v", err)
	}

	db := mongoC.Database(dbName)

	publisherBroker, err := kafka.NewBroker(kafka.Config{
		Brokers: []string{kafkaURL},
		Writer: kafka.WriterConfig{
			BatchSize:    100,
			WriteTimeout: 5 * time.Second,
		},
	})
	if err != nil {
		log.Fatalf("Failed to create kafka broker: %v", err)
	}

	registry := loadSchemaRegistry(schemaRegistryPath)
	router := event.NewStaticRouter(registry)

	publisher := event.NewPublisher(
		router,
		map[string]event.Broker{"kafka": publisherBroker},
		&event.PublisherConfig{
			RetryConfig: &event.RetryConfig{
				InitialInterval: 500 * time.Millisecond,
				MaxElapsedTime:  30 * time.Second,
			},
		})

	repos := repository.NewRepositories(db)
	g, ctx := errgroup.WithContext(ctx)

	auditSvc := service.NewAuditServiceServer(repos.Audit)
	approvalSvc := service.NewApprovalServiceServer(repos.Approvals, repos.ApprovalConfigs, publisher)
	actionSvc := service.NewActionServiceServer(repos.Executions, repos.ActionSchemas, publisher)
	ticketSvc := service.NewTicketServiceServer(repos.Tickets, repos.TicketTypes, publisher)

	// Initialize subscribers
	approvalSubBroker, err := kafka.NewBroker(kafka.Config{
		Brokers: []string{kafkaURL},
		Writer: kafka.WriterConfig{
			BatchSize:    10,
			WriteTimeout: 5 * time.Second,
		},
		Reader: kafka.ReaderConfig{
			GroupID: "approval-service",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create approval subscriber broker: %v", err)
	}

	actionSubBroker, err := kafka.NewBroker(kafka.Config{
		Brokers: []string{kafkaURL},
		Writer: kafka.WriterConfig{
			BatchSize:    10,
			WriteTimeout: 5 * time.Second,
		},
		Reader: kafka.ReaderConfig{
			GroupID: "action-service",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create action subscriber broker: %v", err)
	}

	ticketSubBroker, err := kafka.NewBroker(kafka.Config{
		Brokers: []string{kafkaURL},
		Writer: kafka.WriterConfig{
			BatchSize:    10,
			WriteTimeout: 5 * time.Second,
		},
		Reader: kafka.ReaderConfig{
			GroupID: "ticket-service",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create ticket subscriber broker: %v", err)
	}

	approvalSub := event.NewSubscriber(router, map[string]event.Broker{"kafka": approvalSubBroker}, nil)
	approvalSvc.RegisterHandlers(approvalSub)

	actionSub := event.NewSubscriber(router, map[string]event.Broker{"kafka": actionSubBroker}, nil)
	actionSvc.RegisterHandlers(actionSub)

	ticketSub := event.NewSubscriber(router, map[string]event.Broker{"kafka": ticketSubBroker}, nil)
	ticketSvc.RegisterHandlers(ticketSub)

	// Start subscribers in the background
	subs := []event.Subscriber{approvalSub, actionSub, ticketSub}
	for _, s := range subs {
		sub := s
		g.Go(func() error {
			errCh, err := sub.Start(ctx)
			if err != nil {
				return err
			}
			for err := range errCh {
				log.Printf("Subscriber runtime error: %v", err)
			}
			return nil
		})
	}

	jwtValidator := middleware.NewJwtTokenValidator(jwtSecret)

	grpcMetrics := grpc_prometheus.NewServerMetrics()

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcMetrics.UnaryServerInterceptor(),
			middleware.UnaryLoggingInterceptor(),
			middleware.UnaryRecoveryInterceptor(),
			middleware.UnaryAuthInterceptor(jwtValidator),
		),
		grpc.ChainStreamInterceptor(
			grpcMetrics.StreamServerInterceptor(),
		),
	)
	apiv1.RegisterTicketServiceServer(grpcServer, ticketSvc)
	apiv1.RegisterActionServiceServer(grpcServer, actionSvc)
	apiv1.RegisterAuditServiceServer(grpcServer, auditSvc)
	apiv1.RegisterApprovalServiceServer(grpcServer, approvalSvc)

	grpcMetrics.InitializeMetrics(grpcServer)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("TicketService", grpc_health_v1.HealthCheckResponse_SERVING)

	gwmux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			if strings.ToLower(key) == "authorization" {
				return "authorization", true
			}
			return runtime.DefaultHeaderMatcher(key)
		}),
	)
	addr := ":" + port
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return net.Dial("tcp", lis.Addr().String())
		}),
	}

	if err := apiv1.RegisterTicketServiceHandlerFromEndpoint(ctx, gwmux, lis.Addr().String(), opts); err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}
	if err := apiv1.RegisterActionServiceHandlerFromEndpoint(ctx, gwmux, lis.Addr().String(), opts); err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}
	if err := apiv1.RegisterAuditServiceHandlerFromEndpoint(ctx, gwmux, lis.Addr().String(), opts); err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}
	if err := apiv1.RegisterApprovalServiceHandlerFromEndpoint(ctx, gwmux, lis.Addr().String(), opts); err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/", gwmux)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	origins := strings.Split(corsOrigins, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	corsHandler := c.Handler(mux)

	mixedHandler := grpcHandlerFunc(grpcServer, corsHandler)
	h2cHandler := h2c.NewHandler(mixedHandler, &http2.Server{})

	srv := &http.Server{
		Handler: h2cHandler,
	}

	go func() {
		log.Printf("Starting Multiplexed gRPC & HTTP server on %s", addr)
		if err := srv.Serve(lis); err != nil && err != http.ErrServerClosed {
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
		log.Printf("Server forced to shutdown: %v", err)
	}

	if err := g.Wait(); err != nil {
		log.Printf("Error group finished with error: %v", err)
	}

	log.Println("Server exiting")
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func loadSchemaRegistry(path string) event.SchemaRegistry {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read schema config: %v", err)
	}

	var registry event.SchemaRegistry
	if err := yaml.Unmarshal(data, &registry); err != nil {
		log.Fatalf("Failed to unmarshal schema config: %v", err)
	}

	return registry
}

func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	})
}
