package memdb

import (
	"sync"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/route"
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"go.uber.org/zap"
)

type RoutesStorage interface {
	GetRoutes() ([]*route.Route, error)
	GetUpdatedRoutes(updatedAt int64) ([]*route.Route, error)
}

type RoutesMemDb struct {
	external    RoutesStorage
	lastUpdated int64
	pathToRoute map[string]*route.Route
	lock        sync.RWMutex
	done        chan bool
	interval    time.Duration
	log         *zap.Logger
}

func NewRoutesMemDb(ex RoutesStorage, log *zap.Logger, interval time.Duration) (*RoutesMemDb, error) {
	pathToRoute := map[string]*route.Route{}

	routes, err := ex.GetRoutes()
	if err != nil {
		return nil, err
	}

	numberOfRoutes := 0
	var latetest int64 = -1
	for _, r := range routes {
		pathToRoute[r.Path] = r
		numberOfRoutes++
		if r.UpdatedAt > latetest {
			latetest = r.UpdatedAt
		}
	}

	if numberOfRoutes != 0 {
		log.Sugar().Infof("routes memdb updated at %d with %d routes", latetest, numberOfRoutes)
	}

	return &RoutesMemDb{
		external:    ex,
		pathToRoute: pathToRoute,
		log:         log,
		lastUpdated: latetest,
		interval:    interval,
		done:        make(chan bool),
	}, nil
}

func (mdb *RoutesMemDb) GetRoute(path string) *route.Route {
	r, ok := mdb.pathToRoute[path]
	if ok {
		return r
	}

	return nil
}

func (mdb *RoutesMemDb) SetRoute(r *route.Route) {
	mdb.lock.RLock()
	defer mdb.lock.RUnlock()

	mdb.pathToRoute[r.Path] = r
}

func (mdb *RoutesMemDb) Listen() {
	ticker := time.NewTicker(mdb.interval)
	mdb.log.Info("routes memdb started listening for route updates")

	go func() {
		lastUpdated := mdb.lastUpdated
		for {
			select {
			case <-mdb.done:
				mdb.log.Info("routes memdb stopped")
				return
			case <-ticker.C:
				routes, err := mdb.external.GetUpdatedRoutes(lastUpdated)
				if err != nil {
					stats.Incr("bricksllm.memdb.routes_memdb.listen.get_updated_routes_error", nil, 1)

					mdb.log.Sugar().Debugf("memdb failed to get routes: %v", err)
					continue
				}

				if len(routes) == 0 {
					continue
				}

				any := false
				numberOfUpdated := 0
				for _, r := range routes {
					if r.UpdatedAt > lastUpdated {
						lastUpdated = r.UpdatedAt
					}

					existing := mdb.GetRoute(r.Path)
					if existing == nil || r.UpdatedAt > existing.UpdatedAt {
						mdb.log.Sugar().Infof("routes memdb updated a route: %s", r.Path)
						numberOfUpdated += 1
						any = true
						mdb.SetRoute(r)
					}
				}

				if any {
					mdb.log.Sugar().Infof("routes memdb updated at %d with %d routes", lastUpdated, numberOfUpdated)
				}
			}
		}
	}()
}

func (mdb *RoutesMemDb) Stop() {
	mdb.log.Info("shutting down routes memdb...")

	mdb.done <- true
}
