package proxy

import (
	"encoding/json"

	"github.com/bricks-cloud/bricksllm/internal/provider/openai"
	goopenai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func logCreateRunRequest(log *zap.Logger, data []byte, prod, private bool, cid string) {
	rr := &goopenai.RunRequest{}
	err := json.Unmarshal(data, rr)
	if err != nil {
		logError(log, "error when unmarshalling create run request", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("assistant_id", rr.AssistantID),
			zap.String("model", rr.Model),
			zap.Any("tools", rr.Tools),
			zap.Any("metadata", rr.Metadata),
		}

		if !private {
			fields = append(fields, zap.String("instruction", rr.Instructions))
		}

		log.Info("openai create run request", fields...)
	}
}

func logRunResponse(log *zap.Logger, data []byte, prod, private bool, cid string) {
	r := &goopenai.Run{}
	err := json.Unmarshal(data, r)
	if err != nil {
		logError(log, "error when unmarshalling run response", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("id", r.ID),
			zap.String("object", r.Object),
			zap.Int64("created_at", r.CreatedAt),
			zap.String("thread_id", r.ThreadID),
			zap.String("assistant_id", r.AssistantID),
			zap.String("status", string(r.Status)),
			zap.Any("required_action", r.RequiredAction),
			zap.Any("last_error", r.LastError),
			zap.Int64("expires_at", r.ExpiresAt),
			zap.Int64p("started_at", r.StartedAt),
			zap.Int64p("cancelled_at", r.CancelledAt),
			zap.Int64p("failed_at", r.FailedAt),
			zap.Int64p("completed_at", r.CompletedAt),
			zap.String("model", r.Model),
			zap.Any("tools", r.Tools),
			zap.Any("file_ids", r.FileIDS),
			zap.Any("metadata", r.Metadata),
		}

		if !private && len(r.Instructions) != 0 {
			fields = append(fields, zap.String("instructions", r.Instructions))
		}

		log.Info("openai run response", fields...)
	}
}

func logRetrieveRunRequest(log *zap.Logger, prod bool, cid, tid, rid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("thread_id", tid),
			zap.String("run_id", rid),
		}

		log.Info("openai retrieve run request", fields...)
	}
}

func logModifyRunRequest(log *zap.Logger, data []byte, prod bool, cid, tid, rid string) {
	rr := &goopenai.RunRequest{}
	err := json.Unmarshal(data, rr)
	if err != nil {
		logError(log, "error when unmarshalling modify run request", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("thread_id", tid),
			zap.String("run_id", rid),
			zap.Any("metadata", rr.Metadata),
		}

		log.Info("openai modify run request", fields...)
	}
}

func logListRunsRequest(log *zap.Logger, prod bool, cid, tid string, params map[string]string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("thread_id", tid),
		}

		if v, ok := params["limit"]; ok {
			fields = append(fields, zap.String("limit", v))
		}

		if v, ok := params["order"]; ok {
			fields = append(fields, zap.String("order", v))
		}

		if v, ok := params["after"]; ok {
			fields = append(fields, zap.String("after", v))
		}

		if v, ok := params["before"]; ok {
			fields = append(fields, zap.String("before", v))
		}

		log.Info("openai list runs request", fields...)
	}
}

func logListRunsResponse(log *zap.Logger, data []byte, prod, private bool, cid string) {
	r := &goopenai.RunList{}
	err := json.Unmarshal(data, r)
	if err != nil {
		logError(log, "error when unmarshalling list runs response", prod, err)
		return
	}

	if private {
		for _, run := range r.Runs {
			run.Instructions = ""
		}
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.Any("runs", r.Runs),
		}

		log.Info("openai list runs response", fields...)
	}
}

func logSubmitToolOutputsRequest(log *zap.Logger, data []byte, prod bool, cid, tid, rid string) {
	r := &goopenai.SubmitToolOutputsRequest{}
	err := json.Unmarshal(data, r)
	if err != nil {
		logError(log, "error when unmarshalling submit tool outputs request", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("thread_id", tid),
			zap.String("run_id", rid),
			zap.Any("tool_outputs", r.ToolOutputs),
		}

		log.Info("openai submit tool outputs request", fields...)
	}
}

func logCancelARunRequest(log *zap.Logger, prod bool, cid, tid, rid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("thread_id", tid),
			zap.String("run_id", rid),
		}

		log.Info("openai cancel a run request", fields...)
	}
}

func logCreateThreadAndRunRequest(log *zap.Logger, data []byte, prod, private bool, cid string) {
	r := &openai.CreateThreadAndRunRequest{}
	err := json.Unmarshal(data, r)
	if err != nil {

		logError(log, "error when unmarshalling create thread and run request", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("assistant_id", r.AssistantID),
			zap.String("model", r.Model),
			zap.Any("tools", r.Tools),
			zap.Any("metadata", r.Metadata),
		}

		if !private {
			fields = append(fields, zap.String("instructions", r.Instructions), zap.Any("thread", r.Thread))
		}

		log.Info("openai create thread and run request", fields...)
	}
}

func logRetrieveRunStepRequest(log *zap.Logger, prod bool, cid, tid, rid, sid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("thread_id", tid),
			zap.String("run_id", rid),
			zap.String("step_id", sid),
		}

		log.Info("openai retrieve run step request", fields...)
	}
}

func logRetrieveRunStepResponse(log *zap.Logger, data []byte, prod bool, cid string) {
	rs := &goopenai.RunStep{}
	err := json.Unmarshal(data, rs)
	if err != nil {
		logError(log, "error when unmarshalling retrieve run step response", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("id", rs.ID),
			zap.String("object", rs.Object),
			zap.Int64("created_at", rs.CreatedAt),
			zap.String("assistant_id", rs.AssistantID),
			zap.String("thread_id", rs.ThreadID),
			zap.String("run_id", rs.RunID),
			zap.String("type", string(rs.Type)),
			zap.String("status", string(rs.Status)),
			zap.Any("step_details", rs.StepDetails),
			zap.Any("last_error", rs.LastError),
			zap.Int64p("expired_at", rs.ExpiredAt),
			zap.Int64p("cancelled_at", rs.CancelledAt),
			zap.Int64p("failed_at", rs.FailedAt),
			zap.Int64p("completed_at", rs.CompletedAt),
			zap.Any("metadata", rs.Metadata),
		}

		log.Info("openai retrieve run step request", fields...)
	}
}

func logListRunStepsRequest(log *zap.Logger, prod bool, cid, tid, rid string, params map[string]string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("thread_id", tid),
			zap.String("run_id", rid),
		}

		if v, ok := params["limit"]; ok {
			fields = append(fields, zap.String("limit", v))
		}

		if v, ok := params["order"]; ok {
			fields = append(fields, zap.String("order", v))
		}

		if v, ok := params["after"]; ok {
			fields = append(fields, zap.String("after", v))
		}

		if v, ok := params["before"]; ok {
			fields = append(fields, zap.String("before", v))
		}

		log.Info("openai list run steps request", fields...)
	}
}

func logListRunStepsResponse(log *zap.Logger, data []byte, prod bool, cid string) {
	rsl := &goopenai.RunStepList{}
	err := json.Unmarshal(data, rsl)
	if err != nil {
		logError(log, "error when unmarshalling list run steps response", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.Any("run_steps", rsl.RunSteps),
		}

		log.Info("openai list run steps response", fields...)
	}
}
