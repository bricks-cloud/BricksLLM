package event

type Event struct {
	Id             string
	CreatedAt      int64
	OragnizationId string
	KeyId          string
	CostInUsd      float64
	Provider       string
	Model          string
	Status         int
}

type EventDataPoint struct {
	TimeStamp          int64
	NumberOfRequests   int64
	CostInMicroDollars int64
}
