package custom

type Provider struct {
	Id                  string         `json:"id"`
	CreatedAt           int64          `json:"created_at"`
	UpdatedAt           int64          `json:"updated_at"`
	Provider            string         `json:"provider"`
	RouteConfigs        []*RouteConfig `json:"route_configs"`
	AuthenticationParam string         `json:"authentication_param"`
}

type RouteConfig struct {
	Path                             string `json:"path"`
	TargetUrl                        string `json:"target_url"`
	StreamLocation                   string `json:"stream_location"`
	ModelLocation                    string `json:"model_location"`
	RequestPromptLocation            string `json:"request_prompt_location"`
	ResponseCompletionLocation       string `json:"response_completion_location"`
	StreamEndWord                    string `json:"stream_end_word"`
	StreamResponseCompletionLocation string `json:"stream_response_completion_location"`
	StreamMaxEmptyMessages           int    `json:"stream_max_empty_messages"`
}

type UpdateProvider struct {
	UpdatedAt           int64          `json:"updated_at"`
	RouteConfigs        []*RouteConfig `json:"route_configs"`
	AuthenticationParam *string        `json:"authentication_param"`
}
