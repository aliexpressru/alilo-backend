package minio

import (
	"context"

	"github.com/aliexpressru/alilo-backend/internal/app/config"
	af "github.com/aliexpressru/alilo-backend/pkg/model/ammo"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/minio/minio-go/v7"
)

// var logger = zap.S()

func UploadBytes(ctx context.Context, ammo *pb.AmmoFile, finishChan chan bool) (minio.UploadInfo, error) {
	logger.Info(ctx, "--------  UploadBytes")
	logger.Infof(ctx, "UploadInfo Size File: '%v'", int64(len(ammo.AmmoFile)))

	//	Отправляем в канал для загрузки
	config.Get(ctx).FileToUploadChan <- &af.File{
		AmmoFile:    ammo.GetAmmoFile(),
		Name:        ammo.GetName(),
		BucketName:  ammo.GetBucketName(),
		ContentType: ammo.GetContentType(),
		FinishChan:  finishChan,
	}

	uploadInfo := minio.UploadInfo{
		Size:   ammo.GetSize(),
		Bucket: ammo.GetBucketName(),
		Key:    ammo.GetName(),
	}
	ammo.AmmoFile = []byte{} // если нужен пример валидной строки с байтами, закоментировать(тогда он останется в мапе).

	return uploadInfo, nil
}
