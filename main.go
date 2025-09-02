package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/aliexpressru/alilo-backend/internal/app/config"
	"github.com/aliexpressru/alilo-backend/internal/app/data"
	"github.com/aliexpressru/alilo-backend/internal/app/datapb"
	"github.com/aliexpressru/alilo-backend/internal/app/job"
	svc "github.com/aliexpressru/alilo-backend/internal/app/service"
	"github.com/aliexpressru/alilo-backend/internal/app/swagger"
	"github.com/aliexpressru/alilo-backend/internal/pkg/agent"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	strUtils "github.com/aliexpressru/alilo-backend/pkg/util/string"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/sourcegraph/conc/pool"

	"github.com/jackc/pgx/v4"
)

var (
	ctx      = undecided.NewContextWithMarker(context.Background(), "_alilo", "")
	execPool = pool.New()
)

func main() {
	ctx = logger.ToContext(context.Background(),
		logger.Logger().With(zap.String("_alilo", strings.ToLower("init_config"))))

	// fixme in internal/app/config/cfg_ol.go env load also
	cfg := &config.Config{}
	log.Printf("try to load env ...")
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	log.Printf("Successfully loaded .env file")
	// Print all environment variables for debugging
	if err := env.Parse(cfg); err != nil {
		log.Fatalf("Failed to parse environment variables: %v", err)
	}

	cfg.UploadToLog(ctx)

	db := initDB(cfg)
	dataStore := data.NewStore(db)

	agents, err := dataStore.GetAllMAgents(ctx)
	mustInit(ctx, err)

	var aHosts = make([]string, 0, len(agents))

	for _, agent := range agents {
		aHosts = append(aHosts, undecided.GetMHost(agent))
	}

	am, err := agent.NewAgentManager(ctx, dataStore, aHosts...)
	mustInit(ctx, errors.Wrapf(err, "app.Alilo"))

	mustInit(ctx, errors.Wrapf(err, "app.Alilo"))

	pp := runningJobs(dataStore, am, cfg)

	// init services
	serviceImpl := svc.New(db, pp, am) //fixme pp - не должен туда передаваться!

	// todo open source refactoring ???
	grpcServer := grpc.NewServer()
	gwMux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames:   true,
				EmitUnpopulated: true, // include empty arrays, zeros, false, etc.
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
		//runtime.WithErrorHandler(yourErrorHandler),
	)

	//// register grpc handlers
	pb.RegisterUploadServiceServer(grpcServer, serviceImpl)
	pb.RegisterCommandServiceServer(grpcServer, serviceImpl)

	// register http handlers
	err = pb.RegisterProjectServiceHandlerServer(ctx, gwMux, serviceImpl)
	if err != nil {
		logger.Errorf(ctx, "failed to register ProjectServiceHandlerServer: %v", err)
	}
	err = pb.RegisterScenarioServiceHandlerServer(ctx, gwMux, serviceImpl)
	if err != nil {
		logger.Errorf(ctx, "failed to register ScenarioServiceHandlerServer: %v", err)
	}
	err = pb.RegisterScriptServiceHandlerServer(ctx, gwMux, serviceImpl)
	if err != nil {
		logger.Errorf(ctx, "failed to register ScriptServiceHandlerServer: %v", err)
	}
	err = pb.RegisterUploadServiceHandlerServer(ctx, gwMux, serviceImpl)
	if err != nil {
		logger.Errorf(ctx, "failed to register UploadServiceHandlerServer: %v", err)
	}
	err = pb.RegisterRunServiceHandlerServer(ctx, gwMux, serviceImpl)
	if err != nil {
		logger.Errorf(ctx, "failed to register RunServiceHandlerServer: %v", err)
	}
	err = pb.RegisterAgentServiceHandlerServer(ctx, gwMux, serviceImpl)
	if err != nil {
		logger.Errorf(ctx, "failed to register AgentServiceHandlerServer: %v", err)
	}
	err = pb.RegisterCommandServiceHandlerServer(ctx, gwMux, serviceImpl)
	if err != nil {
		logger.Errorf(ctx, "failed to register CommandServiceHandlerServer: %v", err)
	}
	err = pb.RegisterSearchServiceHandlerServer(ctx, gwMux, serviceImpl)
	if err != nil {
		logger.Errorf(ctx, "failed to register SearchServiceHandlerServer: %v", err)
	}
	err = pb.RegisterParsingServiceHandlerServer(ctx, gwMux, serviceImpl)
	if err != nil {
		logger.Errorf(ctx, "failed to register ParsingServiceHandlerServer: %v", err)
	}
	err = pb.RegisterS3HandlerServer(ctx, gwMux, serviceImpl)
	if err != nil {
		logger.Errorf(ctx, "failed to register S3HandlerServer: %v", err)
	}

	// Create main mux that combines gRPC-Gateway and Swagger
	mainMux := http.NewServeMux()
	mainMux.Handle("/swagger.json", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		res, err := w.Write(SwaggerJSON)
		if err != nil {
			logger.Errorf(ctx, "failed to write SwaggerJSON: %v", err)
		}
		logger.Infof(ctx, "wrote %d bytes of SwaggerJSON", res)
	}))
	mainMux.Handle("/swagger", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		res, err := w.Write([]byte(swagger.SwaggerHTML))
		if err != nil {
			logger.Errorf(ctx, "failed to write SwaggerHTML: %v", err)
		}
		logger.Infof(ctx, "wrote %d bytes of SwaggerHTML", res)
	}))
	mainMux.Handle("/", gwMux) // gRPC-Gateway endpoints last

	// Создаем HTTP-сервер
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.ServicePortDebug),
		Handler:           mainMux,
		ReadHeaderTimeout: 30 * time.Second, // Защита от Slowloris атаки
	}

	go func() {
		logger.Info(ctx, "starting HTTP server on port 8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Обработка graceful shutdown (опционально)
	// Например, при получении сигнала SIGINT или SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info(ctx, "shutting down server...")
	if err := server.Shutdown(ctx); err != nil {
		logger.Error(ctx, "server shutdown failed", zap.Error(err))
	}
}

func runningJobs(dataStore *data.Store, am *agent.Manager, cfg *config.Config) *job.ProcessorPool {
	logger.Warnf(ctx, "Running uploader")
	execPool.Go(func() {
		job.FileUploader(ctx)
	})
	logger.Warnf(ctx, "FileUploader is running")

	logger.Warnf(ctx, "Running CommandProvider")
	execPool.Go(func() {
		job.CommandProvider(ctx, dataStore)
	})
	logger.Warnf(ctx, "CommandProvider is running")

	pp := job.NewProcessorPool(
		dataStore,
		datapb.NewStore(dataStore),
		am,
	)
	for i := 0; i < cfg.CmdProcessorsCount; i++ {
		logger.Warnf(ctx, "Running StartProcessor")
		execPool.Go(func() {
			pp.StartProcessor(ctx)
		})
		logger.Warnf(ctx, "StartProcessor is running")
	}

	logger.Warnf(ctx, "Running StatisticTracker")
	execPool.Go(func() {
		job.StatisticTracker(ctx, pp)
	})
	logger.Warnf(ctx, "StatisticTracker is running")

	return pp
}

func initDB(cfg *config.Config) *sqlx.DB {
	connCfg, err := pgx.ParseConfig(cfg.PgDSN)
	mustInit(ctx, errors.Wrapf(err, "ParseConfig"))

	connCfg.PreferSimpleProtocol = true // binary protocol of prepared statements

	logger.Warnf(ctx,
		"pgxConnCfg: {Host:%v; Port:%v; Database:%v; User:%v; Password:%v; TLSConfig:%v; ConnectTimeout:%v; "+
			"RuntimeParams:%v; LogLevel:%v; connString:%v; PreferSimpleProtocol:%v}",
		connCfg.Host, connCfg.Port, connCfg.Database, connCfg.User, strUtils.MaskString(connCfg.Password),
		connCfg.TLSConfig, connCfg.ConnectTimeout,
		connCfg.RuntimeParams, connCfg.LogLevel, strUtils.MaskString(connCfg.ConnString()),
		connCfg.PreferSimpleProtocol)

	// Create connection pool with stdlib
	db, err := sqlx.Open("pgx", stdlib.RegisterConnConfig(connCfg))
	mustInit(ctx, errors.Wrapf(err, "sqlx.Open"))

	// Configure connection pool
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetConnMaxIdleTime(time.Minute)
	db.SetConnMaxLifetime(time.Hour)

	// Verify connection
	err = db.Ping()
	if err != nil {
		_ = db.Close()
		mustInit(ctx, errors.Wrapf(err, "db.Ping"))
	}

	return db
}

func mustInit(ctx context.Context, err error) {
	if err != nil {
		logger.Fatalf(ctx, "init failure: %s", err)
	}
}
