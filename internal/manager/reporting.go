package manager

import (
	"errors"
	"fmt"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
	"github.com/bricks-cloud/bricksllm/internal/event"
	"github.com/bricks-cloud/bricksllm/internal/key"
)

type costStorage interface {
	GetCounter(keyId string) (int64, error)
}

type keyStorage interface {
	GetKey(keyId string) (*key.ResponseKey, error)
}

type eventStorage interface {
	GetEvents(customId string, keyIds []string) ([]*event.Event, error)
	GetEventDataPoints(start, end, increment int64, tags, keyIds, customIds []string, filters []string) ([]*event.DataPoint, error)
	GetLatencyPercentiles(start, end int64, tags, keyIds []string) ([]float64, error)
}

type ReportingManager struct {
	es eventStorage
	cs costStorage
	ks keyStorage
}

func NewReportingManager(cs costStorage, ks keyStorage, es eventStorage) *ReportingManager {
	return &ReportingManager{
		cs: cs,
		ks: ks,
		es: es,
	}
}

func (rm *ReportingManager) GetEventReporting(e *event.ReportingRequest) (*event.ReportingResponse, error) {
	dataPoints, err := rm.es.GetEventDataPoints(e.Start, e.End, e.Increment, e.Tags, e.KeyIds, e.CustomIds, e.Filters)
	if err != nil {
		return nil, err
	}

	percentiles, err := rm.es.GetLatencyPercentiles(e.Start, e.End, e.Tags, e.KeyIds)
	if err != nil {
		return nil, err
	}

	if len(percentiles) == 0 {
		return nil, internal_errors.NewNotFoundError("latency percentiles are not found")
	}

	return &event.ReportingResponse{
		DataPoints:        dataPoints,
		LatencyInMsMedian: percentiles[0],
		LatencyInMs99th:   percentiles[1],
	}, nil
}

func (rm *ReportingManager) GetKeyReporting(keyId string) (*key.KeyReporting, error) {
	k, err := rm.ks.GetKey(keyId)
	if err != nil {
		return nil, err
	}

	if k == nil {
		return nil, internal_errors.NewNotFoundError("api key is not found")
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

func (rm *ReportingManager) GetEvent(customId string, keyIds []string) (*event.Event, error) {
	if len(customId) == 0 {
		return nil, errors.New("customId cannot be empty")
	}

	events, err := rm.es.GetEvents(customId, keyIds)
	if err != nil {
		return nil, err
	}

	if len(events) >= 1 {
		return events[0], nil
	}

	return nil, internal_errors.NewNotFoundError(fmt.Sprintf("event is not found for customId: %s", customId))
}

func (rm *ReportingManager) GetEvents(customId string, keyIds []string) ([]*event.Event, error) {
	events, err := rm.es.GetEvents(customId, keyIds)
	if err != nil {
		return nil, err
	}

	return events, nil
}
