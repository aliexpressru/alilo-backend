package upload

import (
	"context"
	"fmt"
	"strings"

	"github.com/aliexpressru/alilo-backend/internal/app/config"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	httpUtils "github.com/aliexpressru/alilo-backend/pkg/util/httputil"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	mathutil "github.com/aliexpressru/alilo-backend/pkg/util/math"
	"github.com/aliexpressru/alilo-backend/pkg/util/minio"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
)

// var logger = zap.S()

// FIXME: Не хранить в памяти
// map[scriptID]*pb.AmmoFile
var ammoFiles = make(
	map[int32]*pb.AmmoFile,
	config.Get(undecided.NewContextWithMarker(context.Background(), "_upload", "")).MakeAmmoFilesMap,
)

func AmmoToUpload(ctx context.Context, name string, data *[]byte,
	bucketName string, description string, projectTitle string, scenarioTitle string, contentType string) (
	ammoPoint *pb.AmmoFile, err error) {
	if contentType == "" {
		contentType = "application/json"
	}

	ammoPoint, err = FileToUpload(ctx,
		undecided.AppendDateTime(name),
		data,
		bucketName,
		description,
		projectTitle,
		scenarioTitle,
		contentType,
		nil,
	)
	if err != nil {
		logger.Errorf(ctx, "AmmoToUpload error: '%+v'", err)

		return ammoPoint, err
	}

	ammoFiles[ammoPoint.AmmoId] = ammoPoint
	logger.Infof(ctx, "AmmoToUpload URL: '%v'", ammoPoint.S3Url)

	return ammoPoint, err
}

func ScriptToUpload(ctx context.Context,
	name string, data *[]byte, bucketName string, description string, projectTitle string, scenarioTitle string,
	finishChan chan bool) (
	scriptJsPoint *pb.AmmoFile, err error) {
	if !strings.HasSuffix(name, ".js") {
		name = fmt.Sprint(name, ".js")
	}

	return FileToUpload(ctx,
		name, data, bucketName, description, projectTitle, scenarioTitle, "application/javascript", finishChan)
}

func FileToUpload(ctx context.Context,
	name string, data *[]byte,
	bucketName string, description string, projectTitle string, scenarioTitle string,
	contentType string, finishChan chan bool) (
	filePoint *pb.AmmoFile, err error) {
	if bucketName == "" || bucketName == "-" {
		bucketName = config.Get(ctx).MinioBucket
	}
	logger.Infof(
		ctx,
		"FileToUpload(name:'%v', bucketName:'%v', description:'%v', projectTitle:'%v', scenarioTitle :'%v')",
		name,
		bucketName,
		description,
		projectTitle,
		scenarioTitle,
	)

	name = makeUploadFileName(ctx, projectTitle, scenarioTitle, name)
	logger.Infof(ctx, "FileToUpload file nane: '%v'", name)
	path := makeUploadS3Path(ctx, bucketName, name)
	logger.Infof(ctx, "FileToUpload path: '%v'", path)

	if contentType != "" {
		contentType = "text/plain"
	}

	scheme := httpUtils.SchemeHTTP
	if config.Get(ctx).MinioSecure {
		scheme = httpUtils.SchemeHTTPS
	}

	fileURL := httpUtils.URL(scheme, config.Get(ctx).MinioEndpoint, path)
	fileToUpload := &pb.AmmoFile{
		Name:        name,
		AmmoFile:    *data,
		BucketName:  bucketName,
		Descrip:     description,
		AmmoId:      mathutil.GetRandomID32(ctx),
		S3Url:       fileURL.String(),
		Size:        int64(len(*data)),
		ContentType: contentType,
	}

	_, err = minio.UploadBytes(ctx, fileToUpload, finishChan)
	if err != nil {
		logger.Errorf(ctx, "FileToUpload error: '%v'", err)
		return fileToUpload, err
	}

	logger.Infof(ctx, "FileToUpload URL: '%v'", fileURL.String())

	return fileToUpload, err
}

func GetDataAmmo(ctx context.Context, ammoID int32) (ammo *pb.AmmoFile, message string) {
	logger.Infof(ctx, "Len(AmmoFiles): '%v'", len(ammoFiles))

	ammo, ok := ammoFiles[ammoID]
	if !ok {
		logger.Warnf(ctx, "The required ammo is missing! Ammo(ammoID-> '%v'; ammo-> '%v';)", ammoID, ammo)
		message = fmt.Sprint("The required ", ammoID, " ammo is missing!")
	} else {
		logger.Debugf(ctx, "Ammo(ammoID-> '%v'; ammo-> '%v';)", ammoID, ammo)
		ammo = ammoFiles[ammoID]
	}

	return ammo, message
}

func GetAllAmmo(ctx context.Context) (ammo []*pb.AmmoFile, message string) {
	logger.Infof(ctx, "Len(AmmoFiles): '%v'", len(ammoFiles))

	for _, file := range ammoFiles {
		ammo = append(ammo, file)
	}

	logger.Infof(ctx, "len(ReturnAmmoFiles): '%v'", len(ammo))
	logger.Debugf(ctx, "ReturnAmmoFiles: '%+v'", ammo)

	return ammo, message
}

func makeUploadFileName(ctx context.Context, projectTitle, scenarioTitle, name string) string {
	logger.Infof(ctx,
		"MakeUploadFileName(projectTitle:'%v', scenarioTitle:'%v', name:'%v')",
		projectTitle, scenarioTitle, name)

	return undecided.MakePath([]string{projectTitle, scenarioTitle, name})
}

func makeUploadS3Path(ctx context.Context, bucketName string, name string) string {
	logger.Infof(ctx,
		"MakeUploadPath(bucketName:'%v',name:'%v'", bucketName, name)
	return undecided.MakePath([]string{bucketName, name})
}
