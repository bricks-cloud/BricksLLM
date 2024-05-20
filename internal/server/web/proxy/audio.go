package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/asticode/go-astisub"
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
	goopenai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func getSpeechHandler(prod bool, client http.Client, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		stats.Incr("bricksllm.proxy.get_speech_handler.requests", nil, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, c.Request.Method, "https://api.openai.com/v1/audio/speech", c.Request.Body)
		if err != nil {
			logError(log, "error when creating openai http request", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create openai http request")
			return
		}

		copyHttpHeaders(c.Request, req)

		start := time.Now()

		res, err := client.Do(req)
		if err != nil {
			stats.Incr("bricksllm.proxy.get_speech_handler.http_client_error", nil, 1)

			logError(log, "error when sending create speech request to openai", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send create speech request to openai")
			return
		}
		defer res.Body.Close()

		dur := time.Since(start)
		stats.Timing("bricksllm.proxy.get_speech_handler.latency", dur, nil, 1)

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			logError(log, "error when reading openai create speech response body", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai create speech response body")
			return
		}

		if res.StatusCode == http.StatusOK {
			stats.Incr("bricksllm.proxy.get_speech_handler.success", nil, 1)
			stats.Timing("bricksllm.proxy.get_pass_through_handler.success_latency", dur, nil, 1)
		}

		if res.StatusCode != http.StatusOK {
			stats.Timing("bricksllm.proxy.get_speech_handler.error_latency", dur, nil, 1)
			stats.Incr("bricksllm.proxy.get_speech_handler.error_response", nil, 1)

			errorRes := &goopenai.ErrorResponse{}
			err = json.Unmarshal(bytes, errorRes)
			if err != nil {
				logError(log, "error when unmarshalling openai create speech error response body", prod, err)
			}

			logOpenAiError(log, prod, errorRes)
		}

		for name, values := range res.Header {
			for _, value := range values {
				c.Header(name, value)
			}
		}

		c.Data(res.StatusCode, res.Header.Get("Content-Type"), bytes)
	}
}

func convertVerboseJson(resp *goopenai.AudioResponse, format string) ([]byte, error) {
	if format == "verbose_json" || format == "json" {
		selected := resp
		if format == "json" {
			selected = &goopenai.AudioResponse{
				Text: resp.Text,
			}
		}

		data, err := json.Marshal(selected)
		if err != nil {
			return nil, err
		}

		return data, nil
	}

	if format == "text" {
		return []byte(resp.Text + "\n"), nil
	}

	if format == "srt" || format == "vtt" {
		sub := astisub.NewSubtitles()
		items := []*astisub.Item{}

		for _, seg := range resp.Segments {
			item := &astisub.Item{
				StartAt: time.Duration(seg.Start * float64(time.Second)),
				EndAt:   time.Duration(seg.End * float64(time.Second)),
				Lines: []astisub.Line{
					{
						Items: []astisub.LineItem{
							{Text: seg.Text},
						},
					},
				},
			}

			items = append(items, item)
		}

		sub.Items = items

		buf := bytes.NewBuffer([]byte{})

		if format == "srt" {
			err := sub.WriteToSRT(buf)
			if err != nil {
				return nil, err
			}

			return buf.Bytes(), nil
		}

		if format == "vtt" {
			err := sub.WriteToWebVTT(buf)
			if err != nil {
				return nil, err
			}

			return buf.Bytes(), nil
		}
	}

	return nil, nil
}

func getContentType(format string) string {
	if format == "verbose_json" || format == "json" {
		return "application/json"
	}

	return "text/plain; charset=utf-8"
}

func getTranscriptionsHandler(prod bool, client http.Client, timeOut time.Duration, e estimator) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		stats.Incr("bricksllm.proxy.get_transcriptions_handler.requests", nil, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, c.Request.Method, "https://api.openai.com/v1/audio/transcriptions", c.Request.Body)
		if err != nil {
			logError(log, "error when creating transcriptions openai http request", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create openai transcriptions http request")
			return
		}

		copyHttpHeaders(c.Request, req)

		var b bytes.Buffer
		writer := multipart.NewWriter(&b)

		err = writeFieldToBuffer([]string{
			"model",
			"language",
			"prompt",
			"response_format",
			"temperature",
		}, c, writer, map[string]string{
			"response_format": "verbose_json",
		})
		if err != nil {
			stats.Incr("bricksllm.proxy.get_transcriptions_handler.write_field_to_buffer_error", nil, 1)
			logError(log, "error when writing field to buffer", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot write field to buffer")
			return
		}

		var form TransriptionForm
		c.ShouldBind(&form)

		if form.File != nil {
			fieldWriter, err := writer.CreateFormFile("file", form.File.Filename)
			if err != nil {
				stats.Incr("bricksllm.proxy.get_transcriptions_handler.create_transcription_file_error", nil, 1)
				logError(log, "error when creating transcription file", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot create transcription file")
				return
			}

			opened, err := form.File.Open()
			if err != nil {
				stats.Incr("bricksllm.proxy.get_transcriptions_handler.open_transcription_file_error", nil, 1)
				logError(log, "error when openning transcription file", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot open transcription file")
				return
			}

			_, err = io.Copy(fieldWriter, opened)
			if err != nil {
				stats.Incr("bricksllm.proxy.get_transcriptions_handler.copy_transcription_file_error", nil, 1)
				logError(log, "error when copying transcription file", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot copy transcription file")
				return
			}
		}

		req.Header.Set("Content-Type", writer.FormDataContentType())
		writer.Close()
		req.Body = io.NopCloser(&b)

		start := time.Now()

		res, err := client.Do(req)
		if err != nil {
			stats.Incr("bricksllm.proxy.get_transcriptions_handler.http_client_error", nil, 1)

			logError(log, "error when sending transcriptions request to openai", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send transcriptions request to openai")
			return
		}
		defer res.Body.Close()

		dur := time.Since(start)
		stats.Timing("bricksllm.proxy.get_transcriptions_handler.latency", dur, nil, 1)

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			logError(log, "error when reading openai transcriptions response body", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai transcriptions response body")
			return
		}

		format := c.PostForm("response_format")
		for name, values := range res.Header {
			for _, value := range values {
				if strings.ToLower(name) == "content-type" {
					c.Header(name, getContentType(format))
					continue
				}

				c.Header(name, value)
			}
		}

		if res.StatusCode == http.StatusOK {
			stats.Incr("bricksllm.proxy.get_transcriptions_handler.success", nil, 1)
			stats.Timing("bricksllm.proxy.get_transcriptions_handler.success_latency", dur, nil, 1)

			ar := &goopenai.AudioResponse{}
			err = json.Unmarshal(bytes, ar)
			if err != nil {
				logError(log, "error when unmarshalling openai http audio response body", prod, err)
			}

			if err == nil {
				cost, err := e.EstimateTranscriptionCost(ar.Duration, c.GetString("model"))
				if err != nil {
					stats.Incr("bricksllm.proxy.get_transcriptions_handler.estimate_total_cost_error", nil, 1)
					logError(log, "error when estimating openai cost", prod, err)
				}

				c.Set("costInUsd", cost)
			}

			data, err := convertVerboseJson(ar, format)
			if err != nil {
				stats.Incr("bricksllm.proxy.get_transcriptions_handler.convert_verbose_json_error", nil, 1)
				logError(log, "error when converting verbose json", prod, err)
			}

			c.Header("Content-Length", strconv.Itoa(len(data)))

			c.Data(res.StatusCode, getContentType(format), data)
			return
		}

		if res.StatusCode != http.StatusOK {
			stats.Timing("bricksllm.proxy.get_transcriptions_handler.error_latency", dur, nil, 1)
			stats.Incr("bricksllm.proxy.get_transcriptions_handler.error_response", nil, 1)

			errorRes := &goopenai.ErrorResponse{}
			err = json.Unmarshal(bytes, errorRes)
			if err != nil {
				logError(log, "error when unmarshalling openai transcriptions error response body", prod, err)
			}

			logOpenAiError(log, prod, errorRes)

			c.Data(res.StatusCode, res.Header.Get("Content-Type"), bytes)

			return
		}
	}
}

func getTranslationsHandler(prod bool, client http.Client, timeOut time.Duration, e estimator) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		stats.Incr("bricksllm.proxy.get_translations_handler.requests", nil, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, c.Request.Method, "https://api.openai.com/v1/audio/translations", c.Request.Body)
		if err != nil {
			logError(log, "error when creating translations openai http request", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create openai translations http request")
			return
		}

		copyHttpHeaders(c.Request, req)

		var b bytes.Buffer
		writer := multipart.NewWriter(&b)

		err = writeFieldToBuffer([]string{
			"model",
			"prompt",
			"response_format",
			"temperature",
		}, c, writer, map[string]string{
			"response_format": "verbose_json",
		})
		if err != nil {
			stats.Incr("bricksllm.proxy.get_pass_through_handler.write_field_to_buffer_error", nil, 1)
			logError(log, "error when writing field to buffer", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot write field to buffer")
			return
		}

		var form TranslationForm
		c.ShouldBind(&form)

		if form.File != nil {
			fieldWriter, err := writer.CreateFormFile("file", form.File.Filename)
			if err != nil {
				stats.Incr("bricksllm.proxy.get_pass_through_handler.create_translation_file_error", nil, 1)
				logError(log, "error when creating translation file", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot create translation file")
				return
			}

			opened, err := form.File.Open()
			if err != nil {
				stats.Incr("bricksllm.proxy.get_pass_through_handler.open_translation_file_error", nil, 1)
				logError(log, "error when openning translation file", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot open translation file")
				return
			}

			_, err = io.Copy(fieldWriter, opened)
			if err != nil {
				stats.Incr("bricksllm.proxy.get_pass_through_handler.copy_translation_file_error", nil, 1)
				logError(log, "error when copying translation file", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot copy translation file")
				return
			}
		}

		req.Header.Set("Content-Type", writer.FormDataContentType())

		writer.Close()

		req.Body = io.NopCloser(&b)

		start := time.Now()

		res, err := client.Do(req)
		if err != nil {
			stats.Incr("bricksllm.proxy.get_translations_handler.http_client_error", nil, 1)

			logError(log, "error when sending translations request to openai", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send translations request to openai")
			return
		}
		defer res.Body.Close()

		dur := time.Since(start)
		stats.Timing("bricksllm.proxy.get_translations_handler.latency", dur, nil, 1)

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			logError(log, "error when reading openai translations response body", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai translations response body")
			return
		}

		format := c.PostForm("response_format")
		for name, values := range res.Header {
			for _, value := range values {
				if strings.ToLower(name) == "content-type" {
					c.Header(name, getContentType(format))
					continue
				}

				c.Header(name, value)
			}
		}

		if res.StatusCode == http.StatusOK {
			stats.Incr("bricksllm.proxy.get_translations_handler.success", nil, 1)
			stats.Timing("bricksllm.proxy.get_translations_handler.success_latency", dur, nil, 1)

			ar := &goopenai.AudioResponse{}
			err = json.Unmarshal(bytes, ar)
			if err != nil {
				logError(log, "error when unmarshalling openai http audio response body", prod, err)
			}

			if err == nil {
				cost, err := e.EstimateTranscriptionCost(ar.Duration, c.GetString("model"))
				if err != nil {
					stats.Incr("bricksllm.proxy.get_translations_handler.estimate_total_cost_error", nil, 1)
					logError(log, "error when estimating openai cost", prod, err)
				}

				c.Set("costInUsd", cost)
			}

			data, err := convertVerboseJson(ar, format)
			if err != nil {
				stats.Incr("bricksllm.proxy.get_translations_handler.convert_verbose_json_error", nil, 1)
				logError(log, "error when converting verbose json", prod, err)
			}

			c.Header("Content-Length", strconv.Itoa(len(data)))

			c.Data(res.StatusCode, getContentType(format), data)
			return
		}

		if res.StatusCode != http.StatusOK {
			stats.Timing("bricksllm.proxy.get_translations_handler.error_latency", dur, nil, 1)
			stats.Incr("bricksllm.proxy.get_translations_handler.error_response", nil, 1)

			errorRes := &goopenai.ErrorResponse{}
			err = json.Unmarshal(bytes, errorRes)
			if err != nil {
				logError(log, "error when unmarshalling openai translations error response body", prod, err)
			}

			logOpenAiError(log, prod, errorRes)

			c.Data(res.StatusCode, res.Header.Get("Content-Type"), bytes)

			return
		}
	}
}

type SpeechRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format"`
	Speed          float64 `json:"speed"`
}

func logCreateSpeechRequest(log *zap.Logger, csr *goopenai.CreateSpeechRequest, prod, private bool) {
	if prod {
		fields := []zapcore.Field{
			zap.String("model", string(csr.Model)),
			zap.String("voice", string(csr.Voice)),
		}

		if !private {
			fields = append(fields, zap.String("input", csr.Input))
		}

		if len(csr.ResponseFormat) != 0 {
			fields = append(fields, zap.String("response_format", string(csr.ResponseFormat)))
		}

		if csr.Speed != 0 {
			fields = append(fields, zap.Float64("speed", csr.Speed))
		}

		log.Info("openai create speech request", fields...)
	}
}

func logCreateTranscriptionRequest(log *zap.Logger, model, language, prompt, responseFormat string, temperature float64, prod, private bool) {
	if prod {
		fields := []zapcore.Field{
			zap.String("model", model),
		}

		if !private && len(prompt) != 0 {
			fields = append(fields, zap.String("prompt", prompt))
		}

		if len(language) != 0 {
			fields = append(fields, zap.String("language", language))
		}

		if len(responseFormat) != 0 {
			fields = append(fields, zap.String("response_format", responseFormat))
		}

		if temperature != 0 {
			fields = append(fields, zap.Float64("temperature", temperature))
		}

		log.Info("openai create transcription request", fields...)
	}
}

func logCreateTranslationRequest(log *zap.Logger, model, prompt, responseFormat string, temperature float64, prod, private bool) {
	if prod {
		fields := []zapcore.Field{
			zap.String("model", model),
			zap.Float64("temperature", temperature),
		}

		if !private && len(prompt) == 0 {
			fields = append(fields, zap.String("prompt", prompt))
		}

		if len(responseFormat) != 0 {
			fields = append(fields, zap.String("response_format", responseFormat))
		}

		log.Info("openai create translation request", fields...)
	}
}
