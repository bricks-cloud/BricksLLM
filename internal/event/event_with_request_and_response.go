package event

import (
	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/provider/custom"
)

type EventWithRequestAndContent struct {
	Event               *Event
	IsEmbeddingsRequest bool
	RouteConfig         *custom.RouteConfig
	Request             interface{}
	Content             string
	Response            interface{}
	Key                 *key.ResponseKey
}
