package logger

type ApiMessage struct {
	ClientIp   string   `json:"clientIp"`
	InstanceId string   `json:"instanceId"`
	Latency    *Latency `json:"latency"`
	CreatedAt  int64    `json:"created_at"`
}

type Latency struct {
	Proxy int `json:"proxy"`
	Atlas int `json:"atlas"`
	Total int `json:"total"`
}

type Route struct {
	Path     string `json:"path"`
	Protocol string `json:"protocol"`
}

type Response struct {
	Headers map[string]string `json:"headers"`
	Status  int               `json:"status"`
	Type    MessageType       `json:"type"`
	Size    int               `json:"size"`
	Route   *Route            `json:"route"`
}
