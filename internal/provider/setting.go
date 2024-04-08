package provider

type Setting struct {
	CreatedAt     int64             `json:"createdAt"`
	UpdatedAt     int64             `json:"updatedAt"`
	Provider      string            `json:"provider"`
	Setting       map[string]string `json:"setting,omitempty"`
	Id            string            `json:"id"`
	Name          string            `json:"name"`
	AllowedModels []string          `json:"allowedModels"`
	CostMap       *CostMap          `json:"costMap"`
}

type CostMap struct {
	PromptCostPerModel     map[string]float64 `json:"promptCostPerModel"`
	CompletionCostPerModel map[string]float64 `json:"completionCostPerModel"`
}

func (s *Setting) GetParam(key string) string {
	return s.Setting[key]
}

type UpdateSetting struct {
	UpdatedAt     int64             `json:"updatedAt"`
	Setting       map[string]string `json:"setting,omitempty"`
	Name          *string           `json:"name"`
	AllowedModels *[]string         `json:"allowedModels,omitempty"`
	CostMap       *CostMap          `json:"costMap,omitempty"`
}
