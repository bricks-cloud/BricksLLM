package key

type KeyReporting struct {
	Id                 string `json:"id"`
	CostInMicroDollars int64  `json:"costInMicroDollars"`
}

type KeyRequest struct {
	KeyIds      []string `json:"keyIds"`
	Tags        []string `json:"tags"`
	Name        string   `json:"name"`
	Revoked     *bool    `json:"revoked"`
	Limit       int      `json:"limit"`
	Offset      int      `json:"offset"`
	Order       string   `json:"order"`
	ReturnCount bool     `json:"returnCount"`
}

type GetKeysResponse struct {
	Keys  []*ResponseKey `json:"keys"`
	Count int            `json:"count"`
}
