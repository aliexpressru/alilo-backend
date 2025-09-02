package processing

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/aliexpressru/alilo-backend/internal/app/config"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/minio/minio-go/v7"
)

type Provider struct {
	client     *minio.Client
	bucketName string
}

func GetS3Provider(ctx context.Context) (*Provider, error) {
	cfg := config.Get(ctx)

	logger.Infof(ctx, "get provider for %s bucket", cfg.MinioBucket)

	return &Provider{
		client:     cfg.MinioClient,
		bucketName: cfg.MinioBucket,
	}, nil
}

func (p *Provider) GetList(ctx context.Context) (*pb.GetBucketListResponse, error) {
	s3Objects := p.client.ListObjects(ctx, p.bucketName, minio.ListObjectsOptions{
		Recursive: true,
	})

	contents := make([]*pb.StorageObject, 0)
	for obj := range s3Objects {
		if err := obj.Err; err != nil {
			return nil, err
		}
		contents = append(contents, &pb.StorageObject{
			Key:          obj.Key,
			LastModified: obj.LastModified.Format(time.RFC3339),
			Etag:         obj.ETag,
			Size:         obj.Size,
			StorageClass: obj.StorageClass,
			Owner: &pb.Owner{
				Id:          obj.Owner.ID,
				DisplayName: obj.Owner.DisplayName,
			},
			Type: obj.ContentType,
		})
	}

	nodes := makeNodes(contents, p.bucketName)

	return &pb.GetBucketListResponse{
		Result:     nodes,
		BucketName: p.bucketName,
	}, nil
}

func (p Provider) UploadFile(
	ctx context.Context,
	name string,
	size int64,
	contentType string,
	reader io.Reader,
) (*pb.UploadFileResponse, error) {
	logger.Infof(ctx, "attempting to upload file to bucket %s with path %s", p.bucketName)
	info, err := p.client.PutObject(ctx, p.bucketName, name, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		logger.Errorf(ctx, "failed to upload %v", err)
		return nil, err
	}
	return &pb.UploadFileResponse{
		Name:         info.Key,
		ETag:         info.ETag,
		Size:         info.Size,
		LastModified: info.LastModified.String(),
	}, nil
}

func (p Provider) DeleteFile(ctx context.Context, path string) error {
	logger.Infof(ctx, "attempting to delete file %s from bucket %s", path, p.bucketName)
	err := p.client.RemoveObject(ctx, p.bucketName, path, minio.RemoveObjectOptions{})
	if err != nil {
		return err
	}
	//here delete file
	return nil
}

func addNode(node *pb.ObjectNode, elements []string, storageObject *pb.StorageObject, level int32, path string) {
	nodePath := path + elements[0] + "/"
	if len(elements) == 1 {
		nodeType := pb.ObjectType_FILE
		if elements[0] == "" {
			nodeType = pb.ObjectType_EMPTY
		}
		node.Children = append(node.Children, &pb.ObjectNode{
			Name:          elements[0],
			StorageObject: storageObject,
			Type:          nodeType,
			Children:      []*pb.ObjectNode{},
			Level:         level,
			Path:          nodePath[:len(nodePath)-1],
		})
		return
	}

	var existNode *pb.ObjectNode
	for _, child := range node.Children {
		if child.Name == elements[0] {
			existNode = child
			break
		}
	}

	if existNode == nil {
		newNode := &pb.ObjectNode{
			Name:          elements[0],
			StorageObject: storageObject,
			Type:          pb.ObjectType_DIR,
			Children:      []*pb.ObjectNode{},
			Level:         level,
			Path:          nodePath,
		}
		node.Children = append(node.Children, newNode)
		existNode = newNode
	}

	splicedElements := elements[1:]
	addNode(existNode, splicedElements, storageObject, level+1, nodePath)
}

func makeNodes(storageObjects []*pb.StorageObject, bucketName string) *pb.ObjectNode {
	rootNode := &pb.ObjectNode{
		Name:       "root",
		Type:       pb.ObjectType_ROOT,
		Children:   []*pb.ObjectNode{},
		Level:      0,
		Path:       "/",
		BucketName: bucketName,
	}

	for _, so := range storageObjects {
		elements := strings.Split(so.Key, "/")

		addNode(rootNode, elements, so, 1, rootNode.Path)
	}

	return rootNode
}
