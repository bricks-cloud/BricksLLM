package apikey

type ResponseApiKey struct {
	UpdatedAt int64
	Provider  string
	Key       string
	Id        string
	KeyName   string
}

type RequestApiKey struct {
	UpdatedAt int64
	Provider  string `json:"provider"`
	Key       string `json:"key"`
	KeyName   string `json:"key_name"`
	Id        string
}
