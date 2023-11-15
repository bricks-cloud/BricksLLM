package event

type DataPoint struct {
	TimeStamp            int64   `json:"timeStamp"`
	NumberOfRequests     int64   `json:"numberOfRequests"`
	CostInUsd            float64 `json:"costInUsd"`
	LatencyInMs          int     `json:"latencyInMs"`
	PromptTokenCount     int     `json:"promptTokenCount"`
	CompletionTokenCount int     `json:"completionTokenCount"`
	SuccessCount         int     `json:"successCount"`
	Model                string  `json:"model"`
	KeyId                string  `json:"keyId"`
}

type ReportingResponse struct {
	DataPoints        []*DataPoint `json:"dataPoints"`
	LatencyInMsMedian float64      `json:"latencyInMsMedian"`
	LatencyInMs99th   float64      `json:"latencyInMs99th"`
}

type ReportingRequest struct {
	KeyIds    []string `json:"keyIds"`
	Tags      []string `json:"tags"`
	Start     int64    `json:"start"`
	End       int64    `json:"end"`
	Increment int64    `json:"increment"`
	Filters   []string `json:"filters"`
}
