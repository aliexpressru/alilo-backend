package job

import (
	"bytes"
	"context"

	"github.com/aliexpressru/alilo-backend/internal/app/config"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
	"github.com/minio/minio-go/v7"
)

const uploaderContextKey = "_ammo_uploader"

func FileUploader(ctx context.Context) {
	ctx = undecided.NewContextWithMarker(ctx, uploaderContextKey, "")
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "FileUploader failed:", err)
		}
	}()
	logger.Infof(ctx, "Started 'FileUploader'")

	cfg := config.Get(ctx)

	for ammo := range cfg.FileToUploadChan {
		logger.Infof(ctx, "--------  FileUploader")

		data := bytes.NewReader(ammo.AmmoFile)
		lenData := int64(len(ammo.AmmoFile))
		logger.Infof(ctx, "File fileName: '%v'; LenData: '%v';", ammo.Name, lenData)

		uploadInfo, err := cfg.MinioClient.PutObject(
			ctx,
			ammo.BucketName,
			ammo.Name,
			data,
			lenData,
			minio.PutObjectOptions{ContentType: ammo.ContentType},
		)
		if err != nil {
			logger.Error(ctx, "Error Upload PutBytesObject: ", err)
			safeDone(ctx, ammo.FinishChan, false)

			continue
		}

		logger.Infof(ctx, "Successfully uploaded info(Size:'%v'; Bucket:'%v'; Key:'%v'; ETag:'%v';)",
			uploadInfo.Size, uploadInfo.Bucket, uploadInfo.Key, uploadInfo.ETag)
		logger.Debugf(ctx, "Upload err:'%+v' info: '%+v'", err, uploadInfo)
		//ammo.AmmoFile = []byte{}
		safeDone(ctx, ammo.FinishChan, true)
	}

	logger.Errorf(ctx, "For range FileToUploadChan interrupted!")
}

func safeDone(ctx context.Context, finishChan chan bool, b bool) {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "FileUpload safeDone failed: '%+v'", err)
		}
	}()
	if finishChan != nil {
		select {
		case finishChan <- b:
			logger.Infof(ctx, "FileUpload safeDone %v", b)
		default:
			logger.Errorf(ctx, "FileUpload safeDone: couldn't send to the channel")
		}
	} else {
		logger.Infof(ctx, "FileUpload safeDone skipped")
	}
}
