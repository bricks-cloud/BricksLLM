package memdb

import (
	"sync"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/policy"
	"github.com/bricks-cloud/bricksllm/internal/route"
	"github.com/bricks-cloud/bricksllm/internal/telemetry"
	"go.uber.org/zap"
)

type RoutesStorage interface {
	GetRoutes() ([]*route.Route, error)
	GetUpdatedRoutes(updatedAt int64) ([]*route.Route, error)
}

type PoliciesStorage interface {
	GetAllPolicies() ([]*policy.Policy, error)
	GetUpdatedPolicies(updatedAt int64) ([]*policy.Policy, error)
}

type RoutesMemDb struct {
	external            RoutesStorage
	ps                  PoliciesStorage
	lastUpdatedPolicies int64
	lastUpdated         int64
	idToPolicy          map[string]*policy.Policy
	pathToRoute         map[string]*route.Route
	lock                sync.RWMutex
	done                chan bool
	interval            time.Duration
	log                 *zap.Logger
}

func NewRoutesMemDb(ex RoutesStorage, ps PoliciesStorage, log *zap.Logger, interval time.Duration) (*RoutesMemDb, error) {
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

	idToPolicy := map[string]*policy.Policy{}
	policies, err := ps.GetAllPolicies()
	if err != nil {
		return nil, err
	}

	numberOfPolicies := 0
	var platetest int64 = -1
	for _, p := range policies {
		idToPolicy[p.Id] = p
		numberOfPolicies++
		if p.UpdatedAt > platetest {
			platetest = p.UpdatedAt
		}
	}

	if numberOfPolicies != 0 {
		log.Sugar().Infof("policies memdb updated at %d with %d policies", platetest, numberOfPolicies)
	}

	return &RoutesMemDb{
		external:            ex,
		ps:                  ps,
		idToPolicy:          idToPolicy,
		pathToRoute:         pathToRoute,
		log:                 log,
		lastUpdated:         latetest,
		lastUpdatedPolicies: platetest,
		interval:            interval,
		done:                make(chan bool),
	}, nil
}

func (mdb *RoutesMemDb) GetRoute(path string) *route.Route {
	r, ok := mdb.pathToRoute[path]
	if ok {
		return r
	}

	return nil
}

func (mdb *RoutesMemDb) GetPolicy(id string) *policy.Policy {
	p, ok := mdb.idToPolicy[id]
	if ok {
		return p
	}

	return nil
}

func (mdb *RoutesMemDb) SetRoute(r *route.Route) {
	mdb.lock.RLock()
	defer mdb.lock.RUnlock()

	mdb.pathToRoute[r.Path] = r
}

func (mdb *RoutesMemDb) SetPolicy(p *policy.Policy) {
	mdb.lock.RLock()
	defer mdb.lock.RUnlock()

	mdb.idToPolicy[p.Id] = p
}

func (mdb *RoutesMemDb) Listen() {
	ticker := time.NewTicker(mdb.interval)
	mdb.log.Info("routes memdb started listening for route updates")

	go func() {
		lastUpdated := mdb.lastUpdated
		plastUpdated := mdb.lastUpdatedPolicies

		for {
			select {
			case <-mdb.done:
				mdb.log.Info("routes memdb stopped")
				return
			case <-ticker.C:
				routes, err := mdb.external.GetUpdatedRoutes(lastUpdated)
				if err != nil {
					telemetry.Incr("bricksllm.memdb.routes_memdb.listen.get_updated_routes_error", nil, 1)

					mdb.log.Sugar().Debugf("memdb failed to get routes: %v", err)
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

				policies, err := mdb.ps.GetUpdatedPolicies(plastUpdated)
				if err != nil {
					telemetry.Incr("bricksllm.memdb.routes_memdb.listen.get_updated_policies_error", nil, 1)

					mdb.log.Sugar().Debugf("memdb failed to get policies: %v", err)
					continue
				}

				pany := false
				pnumberOfUpdated := 0
				for _, p := range policies {
					if p.UpdatedAt > plastUpdated {
						plastUpdated = p.UpdatedAt
					}

					existing := mdb.GetPolicy(p.Id)
					if existing == nil || p.UpdatedAt > existing.UpdatedAt {
						mdb.log.Sugar().Infof("routes memdb updated a policy: %s", p.Id)
						pnumberOfUpdated += 1
						pany = true
						mdb.SetPolicy(p)
					}
				}

				if pany {
					mdb.log.Sugar().Infof("routes memdb updated at %d with %d policies", plastUpdated, pnumberOfUpdated)
				}
			}
		}
	}()
}

func (mdb *RoutesMemDb) Stop() {
	mdb.log.Info("shutting down routes memdb...")

	mdb.done <- true
}
