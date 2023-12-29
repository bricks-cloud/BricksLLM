package provider

type Setting struct {
	CreatedAt     int64             `json:"createdAt"`
	UpdatedAt     int64             `json:"updatedAt"`
	Provider      string            `json:"provider"`
	Setting       map[string]string `json:"setting,omitempty"`
	Id            string            `json:"id"`
	Name          string            `json:"name"`
	AllowedModels []string          `json:"allowedModels"`
}

func (s *Setting) GetParam(key string) string {
	return s.Setting[key]
}

type UpdateSetting struct {
	UpdatedAt     int64             `json:"updatedAt"`
	Setting       map[string]string `json:"setting,omitempty"`
	Name          *string           `json:"name"`
	AllowedModels *[]string         `json:"allowedModels,omitempty"`
}
