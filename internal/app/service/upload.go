package service

import (
	"context"

	"github.com/aliexpressru/alilo-backend/internal/app/processing/upload"
	"github.com/aliexpressru/alilo-backend/internal/app/util"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
)

func (s *Service) AmmoUpload(ctx context.Context, request *pb.AmmoUploadRequest) (*pb.AmmoUploadResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "ammo_upload")

	logger.Infof(
		ctx,
		"--- -- - Successful request AmmoUpload FileName'%v' FileLen:'%v'",
		request.GetName(),
		len(request.GetAmmoFile()),
	)
	ammoFile, err := upload.AmmoToUpload(ctx,
		request.GetName(),
		&request.AmmoFile,
		request.GetBucketName(),
		request.GetDescription(),
		request.GetProjectTitle(),
		request.GetScenarioTitle(),
		request.GetContentType(),
	)

	var message = ""

	if err != nil {
		_ = util.SetModalHeader(ctx)
		message = err.Error()
	}

	return &pb.AmmoUploadResponse{
		Status:   (ammoFile != nil) && (ammoFile.Size == int64(len(request.GetAmmoFile()))) && (message == ""),
		Message:  message,
		AmmoFile: ammoFile,
	}, nil
}

func (s *Service) GetDataAmmo(ctx context.Context, request *pb.GetDataAmmoRequest) (*pb.GetDataAmmoResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_data_ammo")

	logger.Infof(ctx, "Successful request getDataAmmo: '%v'", request.String())

	ammoFile, message := upload.GetDataAmmo(ctx, request.GetAmmoId())
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.GetDataAmmoResponse{
		Status:   ammoFile != nil,
		Message:  message,
		AmmoFile: ammoFile,
	}, nil
}

func (s *Service) GetAllAmmo(ctx context.Context, request *pb.GetAllAmmoRequest) (*pb.GetAllAmmoResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_all_ammo")

	logger.Infof(ctx, "Successful request getDataAmmo: '%v'", request.String())

	ammoFiles, message := upload.GetAllAmmo(ctx)
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.GetAllAmmoResponse{
		Status:    message == "",
		Message:   message,
		AmmoFiles: ammoFiles,
	}, nil
}
