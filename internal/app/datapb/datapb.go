package datapb

import (
	"github.com/aliexpressru/alilo-backend/internal/app/data"
	"github.com/sourcegraph/conc/pool"
)

var execPool = pool.New()

type Store struct {
	db          *data.Store
	cacheGroups *CacheGroups
}

func (s *Store) GetDataStore() *data.Store {
	return s.db
}
func (s *Store) GetCacheGroups() *CacheGroups {
	return s.cacheGroups
}

func NewStore(db *data.Store) *Store {
	return &Store{
		db:          db,
		cacheGroups: NewCacheGroups(db),
	}
}
