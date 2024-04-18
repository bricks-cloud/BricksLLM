package manager

import (
	"strings"

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
	GetEvents(userId, customId string, keyIds []string, start, end int64) ([]*event.Event, error)
	GetEventDataPoints(start, end, increment int64, tags, keyIds, customIds, userIds []string, filters []string) ([]*event.DataPoint, error)
	GetLatencyPercentiles(start, end int64, tags, keyIds []string) ([]float64, error)
	GetAggregatedEventByDayDataPoints(start, end int64, keyIds []string) ([]*event.DataPoint, error)
	GetUserIds(keyId string) ([]string, error)
	GetCustomIds(keyId string) ([]string, error)
	GetTopKeyDataPoints(start, end int64, tags, keyIds []string, order string, limit, offset int) ([]*event.KeyDataPoint, error)
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
	dataPoints, err := rm.es.GetEventDataPoints(e.Start, e.End, e.Increment, e.Tags, e.KeyIds, e.CustomIds, e.UserIds, e.Filters)
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

func (rm *ReportingManager) GetAggregatedEventByDayReporting(e *event.ReportingRequest) (*event.ReportingResponse, error) {
	dataPoints, err := rm.es.GetAggregatedEventByDayDataPoints(e.Start, e.End, e.KeyIds)
	if err != nil {
		return nil, err
	}

	return &event.ReportingResponse{
		DataPoints: dataPoints,
	}, nil
}

func (rm *ReportingManager) GetTopKeyReporting(r *event.KeyReportingRequest) (*event.KeyReportingResponse, error) {
	if r == nil {
		return nil, internal_errors.NewValidationError("key reporting requst cannot be nil")
	}

	for _, tag := range r.Tags {
		if len(tag) == 0 {
			return nil, internal_errors.NewValidationError("key reporting requst tag cannot be empty")
		}
	}

	for _, kid := range r.KeyIds {
		if len(kid) == 0 {
			return nil, internal_errors.NewValidationError("key reporting requst key id cannot be empty")
		}
	}

	if strings.ToUpper(r.Order) != "DESC" && strings.ToUpper(r.Order) != "ASC" {
		return nil, internal_errors.NewValidationError("key reporting request order can only be desc or asc")
	}

	dataPoints, err := rm.es.GetTopKeyDataPoints(r.Start, r.End, r.Tags, r.KeyIds, r.Order, r.Limit, r.Offset)
	if err != nil {
		return nil, err
	}

	return &event.KeyReportingResponse{
		DataPoints: dataPoints,
	}, nil
}

func (rm *ReportingManager) GetCustomIds(keyId string) ([]string, error) {
	return rm.es.GetCustomIds(keyId)
}

func (rm *ReportingManager) GetUserIds(keyId string) ([]string, error) {
	return rm.es.GetUserIds(keyId)
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

func (rm *ReportingManager) GetEvents(userId, customId string, keyIds []string, start, end int64) ([]*event.Event, error) {
	events, err := rm.es.GetEvents(userId, customId, keyIds, start, end)
	if err != nil {
		return nil, err
	}

	return events, nil
}
