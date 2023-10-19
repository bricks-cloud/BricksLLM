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
