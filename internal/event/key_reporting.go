package event

type KeyDataPoint struct {
	KeyId     string  `json:"keyId"`
	CostInUsd float64 `json:"costInUsd"`
}

type KeyReportingResponse struct {
	DataPoints []*KeyDataPoint `json:"dataPoints"`
}

type KeyReportingRequest struct {
	Tags   []string `json:"tags"`
	Start  int64    `json:"start"`
	End    int64    `json:"end"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
}
