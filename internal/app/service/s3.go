package service

import (
	"bytes"
	"context"

	"github.com/aliexpressru/alilo-backend/internal/app/processing"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) GetBucketList(ctx context.Context, _ *pb.GetBucketListRequest) (*pb.GetBucketListResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_bucket_list")

	provider, err := processing.GetS3Provider(ctx)
	if err != nil {
		return nil, err
	}

	res, err := provider.GetList(ctx)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Service) UploadFile(ctx context.Context, request *pb.UploadFileRequest) (*pb.UploadFileResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_bucket_location")

	// Validate required fields
	if request.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "file name is required")
	}
	provider, err := processing.GetS3Provider(ctx)
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(request.GetData())

	file, err := provider.UploadFile(
		ctx,
		request.GetName(),
		reader.Size(),
		request.GetContentType(),
		reader,
	)
	if err != nil {
		return nil, err
	}

	return file, nil

}

func (s *Service) DeleteFile(ctx context.Context, request *pb.DeleteFileRequest) (*pb.DeleteFileResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "delete_file_bucket")
	provider, err := processing.GetS3Provider(ctx)
	if err != nil {
		return nil, err
	}

	err = provider.DeleteFile(ctx, request.GetPath())
	if err != nil {
		return nil, err
	}

	return &pb.DeleteFileResponse{Path: request.GetPath()}, nil
}
