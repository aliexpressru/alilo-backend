package service

import (
	"github.com/jmoiron/sqlx"
	"google.golang.org/grpc/codes"

	"github.com/aliexpressru/alilo-backend/internal/app/data"
	"github.com/aliexpressru/alilo-backend/internal/app/datapb"
	"github.com/aliexpressru/alilo-backend/internal/app/job"
	"github.com/aliexpressru/alilo-backend/internal/pkg/agent"
	"github.com/aliexpressru/alilo-backend/internal/pkg/sterr"
)

const LoggerContextKey = "_service"

// service errors
var (
	ErrNotImplemented = sterr.New(codes.Unimplemented, "unimplemented")
	ErrUnauthorized   = sterr.New(codes.NotFound, "unauthorized")
	ErrTaskNotFound   = sterr.New(codes.NotFound, "task not found")
	InvalidArgument   = sterr.New(codes.InvalidArgument, "InvalidArgument")
)

type Service struct {
	db *sqlx.DB

	data  *data.Store
	store *datapb.Store

	commandProcessor *job.ProcessorPool
	agentManager     *agent.Manager
}

func New(db *sqlx.DB, pp *job.ProcessorPool, am *agent.Manager) *Service {
	return &Service{
		db:               db,
		data:             data.NewStore(db),
		store:            datapb.NewStore(data.NewStore(db)),
		commandProcessor: pp,
		agentManager:     am,
	}
}
