/*
Package config Конфигурация системы
*/
package config

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"

	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	"github.com/aliexpressru/alilo-backend/pkg/model/ammo"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	strUtils "github.com/aliexpressru/alilo-backend/pkg/util/string"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	ServicePrefix = ""
	DriverName    = "postgres"
	EnvDev        = "dev"
	EnvInfra      = "infra"
	EnvStg        = "stg"
	EnvLocal      = "local"
	EnvProd       = "prod"
	SmokeMarker   = "Smoke_test"
	SmokeDuration = "1m"
)

var cfg *Config

type Config struct {
	ENV    string `envconfig:"ENV"`
	Secret string `                env:"SECRET" default:"-"`

	MinioEndpoint        string `env:"MINIO_ENDPOINT"          default:"-"`
	MinioAccessKeyID     string `env:"MINIO_ACCESS_KEY_ID"     default:"-"`
	MinioSecretAccessKey string `env:"MINIO_SECRET_ACCESS_KEY" default:"-"`
	MinioBucket          string `env:"MINIO_BUCKET"            default:"-"`
	MinioSecure          bool   `env:"MINIO_SECURE"            default:"true"`

	ServicePortHTTP         int           `env:"SERVICE_PORT_HTTP"`
	ServicePortGRPC         int           `env:"SERVICE_PORT_GRPC"`
	ServicePortDebug        int           `env:"SERVICE_PORT_DEBUG"`
	GracefulShutdownTimeout time.Duration `env:"GRACEFUL_SHUTDOWN_TIMEOUT"`
	GracefulShutdownDelay   time.Duration `env:"GRACEFUL_SHUTDOWN_DELAY"`
	LogLevel                string        `env:"LOG_LEVEL"`

	SSHKey string `env:"SSH_KEY" default:"-"`

	PrometheusEndpoint string `env:"PROMETHEUS_ENDPOINT"  default:"-"`
	DefaultHeaders     string `env:"DEFAULT_HEADERS"      default:""`
	DefaultQueryParams string `env:"DEFAULT_QUERY_PARAMS" default:""`

	DefaultTag     string `env:"DEFAULT_TAG"      default:"prod"`
	DefaultStepSec string `env:"DEFAULT_STEP_SEC" default:"1"`

	// deprecated
	// fixme: хранить путь к мастер-скрипту в конфиге или переменной среды
	URIMasterScript     string `env:"URI_MASTER_SCRIPT"  json:"URIMasterScript"`
	Hostname            string `env:"hostname"           json:"hostname"        default:"-"`
	TraceGetterHostname string `env:"TRACE_GETTER_HOST"                         default:"-"`
	KubeGWProxyHostname string `env:"KUBE_GW_PROXY_HOST"                        default:"-"`

	VMCProdHostname   string `env:"VMC_PROD_HOST"    default:"-"`
	VMCProdHCHostname string `env:"VMC_PROD_HC_HOST" default:"-"`
	VMCDevHostname    string `env:"VMC_DEV_HOST"     default:"-"`
	VMCStepSec        string `env:"VMC_STEP_SEC"     default:"60"`

	SchedulerHost string `env:"SCHEDULER_HOST" default:"-"`

	MmChannelID      string `env:"MM_CHANNEL_ID"       default:"-"`
	MmTrashChannelID string `env:"MM_TRASH_CHANNEL_ID" default:"-"`
	DataSources      string `env:"DATA_SOURCES"        default:"[]"`

	PgDSN           string `env:"PG_DSN"                 json:"-"`
	DBLoggerEnabled bool   `env:"DB_LOGGER_ENABLED"`
	// deprecated
	DBIdleConns int `env:"DB_IDLE_CONNS"`
	// deprecated
	DBOpenConns    int `env:"DB_OPEN_CONNS"`
	DBMaxIdleConns int `env:"DB_MAX_IDLE_CONNS"               default:"5"`
	DBMaxOpenConns int `env:"DB_MAX_OPEN_CONNS"               default:"50"`
	// deprecated
	DBIdleTime time.Duration `env:"DB_IDLE_TIME"`
	// deprecated
	DBLifetime time.Duration `env:"DB_LIFETIME"`
	// deprecated
	DBConnMaxIdleTime time.Duration `env:"DB_CONNS_MAX_IDLE_TIME"          default:""`
	// deprecated
	DBConnMaxLifetime time.Duration `env:"DB_CONNS_MAX_LIFETIME"           default:""`

	JobCmdProcessorFrequency      time.Duration `env:"JOB_CMD_PROCESSOR_FREQUENCY"      default:"5s"`
	JobStatisticTrackingFrequency time.Duration `env:"JOB_STATISTIC_TRACKING_FREQUENCY" default:"1.5s"`
	JobAgentTrackingFrequency     time.Duration `env:"JOB_AGENT_TRACKING_FREQUENCY"     default:"1.5s"`

	MaksURLsInOneScriptRun           int32 `env:"MAKS_URLS_IN_ONE_SCRIPT_RUN"           default:"9"`
	TestToProdDifferenceOrchesterLog int32 `env:"TEST_TO_PROD_DIFFERENCE_ORCHESTER_LOG" default:"10"`

	CmdProcessorsCount int `env:"CMD_PROCESSORS_COUNT" default:"1"`
	CmdChanLength      int `env:"CMD_CHANEL_LENGTH"    default:"1"`

	MakeAmmoFilesMap    int `env:"MAKE_AMMO_FILES_MAP"    default:"1000"`
	MaxStaticAmmoLength int `env:"MAX_STATIC_AMMO_LENGTH" default:"5000"`

	SendMetricsToPrometheus bool `env:"SEND_METRICS_TO_PROM"`

	MinioClient          *minio.Client          `json:"-"`
	FileToUploadChan     chan *ammo.File        `json:"-"`
	RunReportCollectChan chan *models.RunReport `json:"-"`
	CommandChan          chan *models.Command   `json:"-"`
}

func new(ctx context.Context) {
	cfg = &Config{}
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	log.Printf("Successfully loaded .env file")
	// Print all environment variables for debugging
	if err := env.Parse(cfg); err != nil {
		log.Fatalf("Failed to parse environment variables: %v", err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		logger.Warnf(ctx, "initConfig os.Hostname() ERROR: '%v", err)
	}

	cfg.Hostname = hostname
	cfg.FileToUploadChan = make(chan *ammo.File, 20)
	cfg.RunReportCollectChan = make(chan *models.RunReport, 20)
	cfg.CommandChan = make(chan *models.Command, cfg.CmdChanLength)
	cfg.MinioClient, err = InitMinioCli(ctx,
		cfg.MinioEndpoint, cfg.MinioAccessKeyID, cfg.MinioSecretAccessKey, cfg.MinioBucket, cfg.MinioSecure)
	if err != nil {
		logger.Errorf(ctx, "Failed to create MinIO client: %v", err)
		return
	}

	CheckValueSendMetricsToPromet(ctx)
}

func InitMinioCli(ctx context.Context,
	endpoint string, login string, password string, bucket string, useSSL bool) (*minio.Client, error) {

	// Log initialization with masked credentials
	logger.Infof(ctx, "Connecting to MinIO at: %s", endpoint)
	logger.Infof(ctx, "Login (AccessKeyID): %s", strUtils.MaskString(login))
	logger.Infof(ctx, "Password (SecretAccessKey): %s", strUtils.MaskString(password))

	// Initialize MinIO client
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(login, password, ""), // login = AccessKeyID, password = SecretAccessKey
		Secure: useSSL,
	})
	if err != nil {
		logger.Errorf(ctx, "Failed to create MinIO client: %v", err)
		return nil, fmt.Errorf("MinIO connection failed: %v", err)
	}

	// Verify connection
	if !client.IsOnline() {
		logger.Errorf(ctx, "MinIO client is offline")
		return nil, errors.New("MinIO client is offline")
	}
	logger.Infof(ctx, "MinIO connection successful")

	// Check if bucket exists
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		logger.Errorf(ctx, "Failed to check bucket '%s': %v", bucket, err)
		return client, err
	}

	if !exists {
		logger.Infof(ctx, "Bucket '%s' does not exist, creating...", bucket)
		err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			logger.Errorf(ctx, "Failed to create bucket '%s': %v", bucket, err)
			return nil, fmt.Errorf("bucket creation failed: %v", err)
		}
		logger.Infof(ctx, "Bucket '%s' created successfully", bucket)
	} else {
		logger.Infof(ctx, "Bucket '%s' already exists", bucket)
	}

	logger.Infof(ctx, "Bucket '%s' exists: %v", bucket, exists)

	return client, nil
}

func CheckValueSendMetricsToPromet(ctx context.Context) (send bool) {
	sendMetricsToPrometheus := os.Getenv("SEND_METRICS_TO_PROM")

	var value bool

	if sendMetricsToPrometheus != "" {
		var err error

		value, err = strconv.ParseBool(sendMetricsToPrometheus)
		if err != nil {
			logger.Warnf(ctx, "The value of SendMetricsToPrometheus is incorrectly('%v') set:'%v'",
				sendMetricsToPrometheus, cfg.SendMetricsToPrometheus)
		} else {
			cfg.SendMetricsToPrometheus = value
			logger.Infof(ctx, "The value of SendMetricsToPrometheus set value '%v'", cfg.SendMetricsToPrometheus)
		}
	}

	return cfg.SendMetricsToPrometheus
}

func Get(ctx context.Context) *Config {
	if cfg == nil {
		logger.Debug(ctx, "config from ctx")
		new(ctx)
	}

	return cfg
}

// Mask маскировка всех чувствительных данных конфига
func (cfg *Config) Mask() *Config {
	config := *cfg
	config.MinioAccessKeyID = strUtils.MaskString(cfg.MinioAccessKeyID)
	config.MinioSecretAccessKey = strUtils.MaskString(cfg.MinioSecretAccessKey)
	config.SSHKey = strUtils.MaskString(cfg.SSHKey)
	config.PgDSN = strUtils.MaskString(cfg.PgDSN)
	return &config
}

func (cfg *Config) UploadToLog(ctx context.Context) {
	configToLog := cfg
	if cfg.ENV != EnvLocal {
		configToLog = cfg.Mask()
	}

	logger.Warnf(ctx, "%v MarshalIndent config: '%+v'", cfg.ENV, configToLog)
}
