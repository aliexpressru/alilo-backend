package conv

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"

	//nolint
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
)

func ModelGroupPageToPB(ctx context.Context, mGroup *models.GroupsPage) (
	pbGroups *pb.GroupsPage, err error) {
	pbGroups = &pb.GroupsPage{}
	if mGroup != nil {
		err = ModelToPb(ctx, mGroup, pbGroups)
		if err != nil {
			err = errors.Wrapf(err, "GroupsPage model to pb: '%+v'", err)
			logger.Error(ctx, err)
			return pbGroups, err
		}
		pbGroups.Groups = ModelToPBGroups(ctx, mGroup.Groups)
	} else {
		return nil, fmt.Errorf("model GroupsPage is nil")
	}

	return pbGroups, err
}

func PBGroupsPageToModel(ctx context.Context, pbGroupsPage *pb.GroupsPage) (
	mGroup *models.GroupsPage, err error) {
	mGroup = &models.GroupsPage{}
	if pbGroupsPage != nil {
		message := PbToModel(ctx, mGroup, pbGroupsPage)
		mGroup.Groups = "["
		for _, group := range pbGroupsPage.Groups {
			groupString, er := PbToString(ctx, group)
			if er != nil {
				err = errors.Wrapf(err, "groups model to pb(string): '%+v'", er)
				logger.Error(ctx, err)
			}
			mGroup.Groups += fmt.Sprintf("%v,", groupString)
		}
		mGroup.Groups = fmt.Sprintf("%v%v", mGroup.Groups[:len(mGroup.Groups)-1], "]")
		if message != "" {
			err = fmt.Errorf("pbGroupsPage model to pb: '%+v'", message)
			logger.Error(ctx, err)
		}
	} else {
		return nil, fmt.Errorf("pbGroupsPage is nil")
	}

	return mGroup, nil
}

func ModelToPBGroups(ctx context.Context, mGroup string) (groups []*pb.Group) {
	if mGroup != "" {
		if json.Valid([]byte(mGroup)) {
			jsonDecoder := json.NewDecoder(strings.NewReader(mGroup))

			_, err := jsonDecoder.Token()
			if err != nil {
				logger.Error(ctx, "Groups Err jsonDecoder: ", err)
			}

			for jsonDecoder.More() {
				protoMessage := &pb.Group{}

				err = jsonpb.UnmarshalNext(jsonDecoder, protoMessage)
				if err != nil {
					logger.Errorf(ctx, "Groups UnmarshalNext error: %v", err)
				}

				groups = append(groups, protoMessage)
			}
		} else {
			logger.Errorf(ctx, "Groups Not Valid json: '%v'", mGroup)
			return []*pb.Group{}
		}
	} else {
		logger.Warnf(ctx, "Groups is empty: '%v'", mGroup)
		return []*pb.Group{}
	}

	return groups
}
