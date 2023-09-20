package manager

import (
	"github.com/bricks-cloud/bricksllm/internal/errors"
	"github.com/bricks-cloud/bricksllm/internal/key"
)

type costStorage interface {
	GetCounter(keyId string) (int64, error)
}

type keyStorage interface {
	GetKey(keyId string) (*key.ResponseKey, error)
}

type ReportingManager struct {
	cs costStorage
	ks keyStorage
}

func NewReportingManager(cs costStorage, ks keyStorage) *ReportingManager {
	return &ReportingManager{
		cs: cs,
		ks: ks,
	}
}

func (rm *ReportingManager) GetKeyReporting(keyId string) (*key.KeyReporting, error) {
	k, err := rm.ks.GetKey(keyId)
	if err != nil {
		return nil, err
	}

	if k == nil {
		return nil, errors.NewNotFoundError("api key is not found")
	}

	micros, err := rm.cs.GetCounter(keyId)
	if err != nil {
		return nil, err
	}

	return &key.KeyReporting{
		Id:                 keyId,
		CostInMicroDollars: micros,
	}, err
}
