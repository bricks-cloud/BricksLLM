package key

type KeyReporting struct {
	Id                 string `json:"id"`
	CostInMicroDollars int64  `json:"costInMicroDollars"`
}

type KeyRequest struct {
	KeyIds  []string `json:"keyIds"`
	Tags    []string `json:"tags"`
	Revoked *bool    `json:"revoked"`
	Limit   int      `json:"limit"`
	Offset  int      `json:"offset"`
}
