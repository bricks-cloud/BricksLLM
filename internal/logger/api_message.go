package logger

import (
	"fmt"

	"github.com/fatih/color"
)

type ApiMessage struct {
	ClientIp   string      `json:"clientIp"`
	InstanceId string      `json:"instanceId"`
	Latency    *Latency    `json:"latency"`
	CreatedAt  int64       `json:"created_at"`
	Route      *Route      `json:"route"`
	Response   *Response   `json:"response"`
	Request    *Request    `json:"request"`
	Type       MessageType `json:"type"`
}

type Latency struct {
	Proxy     int64 `json:"proxy"`
	BricksLlm int64 `json:"bricksllm"`
	Total     int64 `json:"total"`
}

type Route struct {
	Path     string `json:"path"`
	Protocol string `json:"protocol"`
}

type Request struct {
	Headers map[string][]string `json:"headers"`
	Size    int64               `json:"size"`
}

type Response struct {
	Headers   map[string][]string `json:"headers"`
	CreatedAt int64               `json:"createdAt"`
	Status    int                 `json:"status"`
	Size      int64               `json:"size"`
}

func NewApiMessage() *ApiMessage {
	return &ApiMessage{
		Latency:  &Latency{},
		Route:    &Route{},
		Request:  &Request{},
		Response: &Response{},
		Type:     ApiMessageType,
	}
}

func colorStatusCode(status int) string {
	green := color.New(color.BgGreen)
	red := color.New(color.BgRed)
	yellow := color.New(color.BgYellow)
	if status >= 500 {
		return red.Sprintf(" %d ", status)
	}

	if status >= 400 {
		return yellow.Sprintf(" %d ", status)
	}

	return green.Sprintf(" %d ", status)
}

func (am *ApiMessage) DevLogContext() string {
	result := "API | "

	if am.Response.Status != 0 {
		result += (colorStatusCode(am.Response.Status) + " |")
	}

	if am.Latency.Total != 0 {
		result += fmt.Sprintf(" %dms |", am.Latency.Total)
	}

	if len(am.Route.Path) != 0 {
		result += fmt.Sprintf(" %s |", am.Route.Path)
	}

	return result
}

func (am *ApiMessage) SetBricksLlmLatency(latency int64) {
	am.Latency.BricksLlm = latency
}

func (am *ApiMessage) SetTotalLatency(latency int64) {
	am.Latency.Total = latency
}

func (am *ApiMessage) SetClientIp(ip string) {
	am.ClientIp = ip
}

func (am *ApiMessage) SetPath(path string) {
	am.Route.Path = path
}

func (am *ApiMessage) SetProtocol(protocol string) {
	am.Route.Protocol = protocol
}

func (am *ApiMessage) SetInstanceId(id string) {
	am.InstanceId = id
}

func (am *ApiMessage) SetRequestHeaders(headers map[string][]string) {
	am.Request.Headers = headers
}

func (am *ApiMessage) SetResponseHeaders(headers map[string][]string) {
	am.Response.Headers = headers
}

func (am *ApiMessage) SetCreatedAt(createdAt int64) {
	am.CreatedAt = createdAt
}

func (am *ApiMessage) SetProxyLatency(latency int64) {
	am.Latency.Proxy = latency
}

func (am *ApiMessage) SetRequestBodySize(size int64) {
	am.Request.Size = size
}

func (am *ApiMessage) SetResponseBodySize(size int64) {
	am.Response.Size = size
}

func (am *ApiMessage) SetResponseStatus(status int) {
	am.Response.Status = status
}

func (am *ApiMessage) SetResponseCreatedAt(createdAt int64) {
	am.Response.CreatedAt = createdAt
}

func (am *ApiMessage) GetProxyLatency() int64 {
	return am.Latency.Proxy
}

type apiLoggerConfig interface {
	GetHideIp() bool
	GetHideHeaders() bool
}

func (am *ApiMessage) ModifyFileds(c apiLoggerConfig) {
	if c.GetHideIp() {
		am.ClientIp = ""
	}

	if c.GetHideHeaders() {
		am.Request.Headers = map[string][]string{}
		am.Response.Headers = map[string][]string{}
	}
}
