package data

import (
	"context"
	"fmt"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/pkg/errors"
)

func (s *Store) SaveMGroupsPage(ctx context.Context, groups *models.GroupsPage) error {
	err := groups.Insert(ctx, s.db, boil.Blacklist(
		models.GroupsPageColumns.DeletedAt,
		models.GroupsPageColumns.PageID,
	))

	logger.Infof(ctx, "Save mGroupsPage:'%v'", groups)

	return err
}

func (s *Store) GetMGroupsPage(ctx context.Context) (groups *models.GroupsPage, err error) {
	group, err := models.GroupsPages(
		qm.OrderBy(fmt.Sprint(models.GroupsPageColumns.PageID, " desc")),
	).One(ctx, s.db)
	if err != nil {
		err = errors.Wrap(err, "Error get mGroupPage")
		logger.Warnf(ctx, err.Error())

		return nil, err
	}

	logger.Infof(ctx, "Get mGroupPage. Prepared model: '%+v'", group)

	return group, err
}

func (s *Store) GetCountGroupsPage(ctx context.Context) (int64, error) {
	count, err := models.GroupsPages().Count(ctx, s.db)
	logger.Infof(ctx, "Get count mGroup:'%v'", count)

	return count, err
}

// CleaningMGroupsPage удаление последних i записей
func (s *Store) CleaningMGroupsPage(ctx context.Context, i int64) error {
	query := "DELETE FROM groups_page WHERE page_id IN (SELECT page_id FROM groups_page ORDER BY created_at ASC LIMIT $1);"

	_, err := s.db.QueryContext(ctx, query, i)
	if err != nil {
		logger.Errorf(ctx, "error exe CleaningMGroupsPage query{LIMIT %v}: %+v", i, err)
	}

	return err
}
