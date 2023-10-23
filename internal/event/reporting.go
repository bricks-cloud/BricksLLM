package event

type DataPoint struct {
	TimeStamp        int64   `json:"timeStamp"`
	NumberOfRequests int64   `json:"numberOfRequests"`
	CostInUsd        float64 `json:"costInUsd"`
}

type ReportingResponse struct {
	DataPoints []*DataPoint `json:"dataPoints"`
}

type ReportingRequest struct {
	KeyIds    []string `json:"keyIds"`
	Tags      []string `json:"tags"`
	Start     int64    `json:"start"`
	End       int64    `json:"end"`
	Increment int64    `json:"increment"`
}
