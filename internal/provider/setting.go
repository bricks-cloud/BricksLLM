package provider

type Setting struct {
	CreatedAt int64             `json:"createdAt"`
	UpdatedAt int64             `json:"updatedAt"`
	Provider  string            `json:"provider"`
	Setting   map[string]string `json:"setting,omitempty"`
	Id        string            `json:"id"`
	Name      string            `json:"name"`
}
