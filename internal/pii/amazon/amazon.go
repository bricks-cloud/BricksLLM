package amazon

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/comprehend"
	"github.com/aws/aws-sdk-go-v2/service/comprehend/types"
	"github.com/bricks-cloud/bricksllm/internal/pii"
	"github.com/bricks-cloud/bricksllm/internal/telemetry"
	"go.uber.org/zap"
)

type Client struct {
	client *comprehend.Client
	rt     time.Duration
	ct     time.Duration
	log    *zap.Logger
}

func NewClient(rt time.Duration, ct time.Duration, log *zap.Logger, region string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ct)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	client := comprehend.NewFromConfig(cfg)

	return &Client{
		client: client,
		rt:     rt,
		ct:     ct,
		log:    log,
	}, nil
}

func (c *Client) detect(content string) (*comprehend.DetectPiiEntitiesOutput, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.rt)
	defer cancel()

	output, err := c.client.DetectPiiEntities(ctx, &comprehend.DetectPiiEntitiesInput{
		LanguageCode: types.LanguageCodeEn,
		Text:         &content,
	})

	if err != nil {
		return nil, err
	}

	return output, nil
}

func (c *Client) Detect(input []string) (*pii.Result, error) {
	var wg sync.WaitGroup

	result := &pii.Result{
		Detections: make([]*pii.Detection, len(input)),
	}

	for index, text := range input {
		wg.Add(1)
		go func(t string, i int) {
			defer wg.Done()
			detection := &pii.Detection{}
			entities := []*pii.Entity{}

			start := time.Now()

			r, err := c.detect(t)
			if err != nil {
				c.log.Debug("error when detecting pii entities", zap.Error(err))
				telemetry.Incr("bricksllm.amazon.detect.error", nil, 1)
				return
			}

			telemetry.Timing("bricksllm.amazon.detect.latency_in_ms", time.Since(start), nil, 1)

			detection.Input = t

			for _, detected := range r.Entities {
				if detected.BeginOffset != nil && detected.EndOffset != nil {
					entities = append(entities, &pii.Entity{
						BeginOffset: int(*detected.BeginOffset),
						EndOffset:   int(*detected.EndOffset),
						Type:        string(detected.Type),
					})
				}
			}

			detection.Entities = entities

			result.Detections[i] = detection

		}(text, index)
	}

	wg.Wait()

	return result, nil

}
