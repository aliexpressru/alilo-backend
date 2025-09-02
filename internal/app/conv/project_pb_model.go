package conv

import (
	"context"
	"fmt"

	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
)

func ModelToPBProject(ctx context.Context, mProject *models.Project) (project *pb.Project, message string) {
	project = &pb.Project{}
	if mProject != nil {
		err := ModelToPb(ctx, *mProject, project)
		if err != nil {
			message = fmt.Sprintf("Project model to pb: '%+v'", err)
			logger.Errorf(ctx, message)

			return nil, message
		}
	} else {
		return nil, "model project is nil"
	}

	return project, message
}

func PBProjectToModel(ctx context.Context, project *pb.Project) (mProject *models.Project, message string) {
	mProject = &models.Project{}
	if project != nil {
		message = PbToModel(ctx, mProject, project)
		if message != "" {
			message = fmt.Sprintf("Project pb to model: '%v'", message)
			logger.Errorf(ctx, message)

			return nil, message
		}
	} else {
		return nil, "pb project is nil"
	}

	return mProject, message
}
