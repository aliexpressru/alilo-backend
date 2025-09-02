package datapb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aliexpressru/alilo-backend/internal/app/conv"
	"github.com/aliexpressru/alilo-backend/internal/app/data"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
)

// CacheGroups представляет собой сам сквозной кеш
type CacheGroups struct {
	mu         sync.RWMutex
	groupsPage *pb.GroupsPage
	//defaultExpiration    time.Duration
	//cleanupInterval      time.Duration
	periodicSendInterval time.Duration
	db                   *data.Store
	saveCh               chan int8
	exist                bool
}

// NewCacheGroups Создание нового экземпляра
func NewCacheGroups(db *data.Store) *CacheGroups {
	c := &CacheGroups{
		groupsPage: &pb.GroupsPage{
			Groups:            []*pb.Group{},
			PreferredUserName: "-",
			UserName:          "-",
		},
		periodicSendInterval: time.Second * 30,
		db:                   db,
		saveCh:               make(chan int8, 10),
	}

	execPool.Go(c.cacheControl)
	return c
}

func (gpc *CacheGroups) cacheControl() {
	var ctx = undecided.NewContextWithMarker(context.TODO(), "_cache_groups", "cacheControl")

	maxCountGroupsPage := int64(20) // fixme: перенести в конфиг
	for range gpc.saveCh {
		execPool.Go(func() {
			defer func() {
				if err := recover(); err != nil {
					logger.Errorf(ctx, "CommandProcessor failed: '%+v'", err)
				}
			}()
			gpc.mu.Lock()
			defer gpc.mu.Unlock()

			mGroupsPage, err := conv.PBGroupsPageToModel(ctx, gpc.groupsPage)
			if err != nil {
				logger.Errorf(ctx, "conversion groupsPage error: %+v", mGroupsPage)

				return
			}
			err = gpc.db.SaveMGroupsPage(ctx, mGroupsPage)
			if err != nil {
				logger.Errorf(ctx, "error SaveMGroupsPage: %+v", err)

				return
			}

			if count, er := gpc.db.GetCountGroupsPage(ctx); er == nil && count > maxCountGroupsPage {
				logger.Infof(ctx, "the table cacheControl has been triggered: %v", count)
				er = gpc.db.CleaningMGroupsPage(ctx, maxCountGroupsPage/10)
				if er != nil {
					logger.Errorf(ctx, "error CleaningMGroupsPage: %+v", er)
				}
			}
			if !gpc.exist {
				gpc.exist = true
			}
		})
	}
}

func (gpc *CacheGroups) ChangeUser(ctx context.Context, preferredUserName string, userName string) error {
	gpc.mu.Lock()
	defer gpc.mu.Unlock()

	gpc.synchronizingTheCacheWithTheDB(ctx)

	gpc.groupsPage.PreferredUserName = preferredUserName
	gpc.groupsPage.UserName = userName
	select {
	case gpc.saveCh <- 1:
	default:
	}

	return nil
}

func (gpc *CacheGroups) UpdateGroups(ctx context.Context, groupsPage *pb.GroupsPage) (*pb.GroupsPage, error) {
	gpc.mu.Lock()
	defer gpc.mu.Unlock()

	//todo: нужно будет под фичу овнера для изменений
	/*if gpc.preferredUserName != "" {
		if gpc.preferredUserName != preferredUserName {
			msg := "a change of the group owner is required" // todo: нужна API-шка для смены\запроса только владельца
			logger.Warn(ctx, msg)
			return gpc.groupsPage, sterr.New(codes.PermissionDenied, msg)
		}
	}*/

	gpc.synchronizingTheCacheWithTheDB(ctx)

	gpc.groupsPage.Groups = groupsPage.Groups
	gpc.groupsPage.PreferredUserName = groupsPage.PreferredUserName
	gpc.groupsPage.UserName = groupsPage.UserName
	select {
	case gpc.saveCh <- 1:
	default:
	}

	return groupsPage, nil
}

func (gpc *CacheGroups) GetGroupsPage(ctx context.Context) (groupsPage *pb.GroupsPage, err error) {
	gpc.mu.RLock()
	defer gpc.mu.RUnlock()

	gpc.synchronizingTheCacheWithTheDB(ctx)

	return gpc.groupsPage, err
}

func (gpc *CacheGroups) GetUserName(ctx context.Context) (userName, preferredUserName string, err error) {
	gpc.mu.RLock()
	defer gpc.mu.RUnlock()

	gpc.synchronizingTheCacheWithTheDB(ctx)

	return gpc.groupsPage.UserName, gpc.groupsPage.PreferredUserName, err
}

func (gpc *CacheGroups) GetGroup(ctx context.Context, groupID int32) (*pb.Group, error) {
	gpc.mu.RLock()
	defer gpc.mu.RUnlock()

	gpc.synchronizingTheCacheWithTheDB(ctx)

	for _, group := range gpc.groupsPage.Groups {
		if group.GroupId == groupID {
			logger.Infof(ctx, "the group{%v} has been received", group.Title)

			return group, nil
		}
	}
	err := fmt.Errorf("the group{%v} was not found", groupID)

	logger.Error(ctx, "group search error: %v", err)
	return nil, err
}

func (gpc *CacheGroups) GetScenarioGroup(ctx context.Context, groupID int32, scenarioID int32) (
	*pb.ScenarioGroup, error) {
	gpc.mu.RLock()
	defer gpc.mu.RUnlock()

	gpc.synchronizingTheCacheWithTheDB(ctx)

	for _, group := range gpc.groupsPage.Groups {
		if group.GroupId == groupID {
			logger.Infof(ctx, "the group{%v:%v} has been received", groupID, group.Title)
			for _, grScenario := range group.ScenariosGroup {
				if grScenario.ScenarioId == scenarioID {
					logger.Infof(ctx, "the grScenario{%v:%v} has been received",
						scenarioID, grScenario.TitleScenario)
					return grScenario, nil
				}
			}
		}
	}
	err := fmt.Errorf("the scenarioGroup{%v:%v} was not found", groupID, scenarioID)

	logger.Error(ctx, "ScenarioGroup search error: %v", err)
	return nil, err
}

// synchronizingTheCacheWithTheDB если кей пустой, проверить наличие данных в базе и выставить их в кэш
func (gpc *CacheGroups) synchronizingTheCacheWithTheDB(ctx context.Context) bool {
	if !gpc.exist {
		mGroupsPage, err := gpc.db.GetMGroupsPage(ctx)
		if err != nil {
			logger.Errorf(ctx, "error getting mGroupsPage: %+v", err)

			return false
		}

		if mGroupsPage != nil {
			pbGroupPage, er := conv.ModelGroupPageToPB(ctx, mGroupsPage)
			if er != nil {
				logger.Errorf(ctx, "%+v", er)

				return false
			}

			if pbGroupPage != nil {
				gpc.groupsPage = pbGroupPage
				gpc.exist = true
			}
		}
	} else {
		logger.Infof(ctx, "the cache does not require synchronization")
	}

	return gpc.exist
}
