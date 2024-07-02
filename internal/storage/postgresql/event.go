package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/bricks-cloud/bricksllm/internal/event"
	"github.com/lib/pq"
)

func (s *Store) CreateEventsByDayTable() error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS event_agg_by_day (
		id SERIAL PRIMARY KEY,
		time_stamp BIGINT NOT NULL,
		num_of_requests INT NOT NULL,
		cost_in_usd FLOAT8 NOT NULL,
		latency_in_ms INT NOT NULL,
		prompt_token_count INT NOT NULL,
		success_count INT NOT NULL,
		completion_token_count INT NOT NULL,
		key_id VARCHAR(255)
	)`

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	_, err := s.db.ExecContext(ctxTimeout, createTableQuery)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) AlterEventsTable() error {
	alterTableQuery := `
		ALTER TABLE events ADD COLUMN IF NOT EXISTS path VARCHAR(255), ADD COLUMN IF NOT EXISTS method VARCHAR(255), ADD COLUMN IF NOT EXISTS custom_id VARCHAR(255), ADD COLUMN IF NOT EXISTS request JSONB, ADD COLUMN IF NOT EXISTS response JSONB, ADD COLUMN IF NOT EXISTS user_id VARCHAR(255) NOT NULL DEFAULT '', ADD COLUMN IF NOT EXISTS action VARCHAR(255) NOT NULL DEFAULT '', ADD COLUMN IF NOT EXISTS policy_id VARCHAR(255) NOT NULL DEFAULT '',  ADD COLUMN IF NOT EXISTS route_id VARCHAR(255) NOT NULL DEFAULT '',  ADD COLUMN IF NOT EXISTS correlation_id VARCHAR(255) NOT NULL DEFAULT '', ADD COLUMN IF NOT EXISTS metadata JSONB;
	`

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	_, err := s.db.ExecContext(ctxTimeout, alterTableQuery)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) CreateUniqueIndexForEventsTable() error {
	createIndexQuery := `
	CREATE UNIQUE index IF NOT EXISTS idx_key_id_and_time_stamp on event_agg_by_day (time_stamp, key_id);`

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	_, err := s.db.ExecContext(ctxTimeout, createIndexQuery)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) CreateTimeStampIndexForEventsTable() error {
	createIndexQuery := `
	CREATE index IF NOT EXISTS idx_time_stamp on event_agg_by_day (time_stamp);`

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	_, err := s.db.ExecContext(ctxTimeout, createIndexQuery)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) CreateKeyIdIndexForEventsTable() error {
	createIndexQuery := `
	CREATE index IF NOT EXISTS idx_key_id on event_agg_by_day (key_id);`

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	_, err := s.db.ExecContext(ctxTimeout, createIndexQuery)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) CreateEventsTable() error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS events (
		event_id VARCHAR(255) PRIMARY KEY,
		created_at BIGINT NOT NULL,
		tags VARCHAR(255)[],
		key_id VARCHAR(255),
		cost_in_usd FLOAT8,
		provider VARCHAR(255),
		model VARCHAR(255),
		status_code INT,
		prompt_token_count INT,
		completion_token_count INT,
		latency_in_ms INT
	)`

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	_, err := s.db.ExecContext(ctxTimeout, createTableQuery)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) GetEvents(userId string, customId string, keyIds []string, start int64, end int64) ([]*event.Event, error) {
	if len(customId) == 0 && len(keyIds) == 0 && len(userId) == 0 {
		return nil, errors.New("none of customId, keyIds and userId is specified")
	}

	if len(keyIds) != 0 && (start == 0 || end == 0) {
		return nil, errors.New("keyIds are provided but either start or end is not specified")
	}

	query := `
		SELECT * FROM events WHERE
	`

	if len(customId) != 0 {
		query += fmt.Sprintf(" custom_id = '%s'", customId)
	}

	if len(customId) > 0 && len(userId) > 0 {
		query += " AND"
	}

	if len(userId) != 0 {
		query += fmt.Sprintf(" user_id = '%s'", userId)
	}

	if (len(customId) > 0 || len(userId) > 0) && len(keyIds) > 0 {
		query += " AND"
	}

	if len(keyIds) != 0 {
		query += fmt.Sprintf(" key_id = ANY('%s') AND created_at >= %d AND created_at <= %d", sliceToSqlStringArray(keyIds), start, end)
	}

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	events := []*event.Event{}
	rows, err := s.db.QueryContext(ctxTimeout, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return events, nil
		}
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var e event.Event
		var path sql.NullString
		var method sql.NullString
		var customId sql.NullString

		if err := rows.Scan(
			&e.Id,
			&e.CreatedAt,
			pq.Array(&e.Tags),
			&e.KeyId,
			&e.CostInUsd,
			&e.Provider,
			&e.Model,
			&e.Status,
			&e.PromptTokenCount,
			&e.CompletionTokenCount,
			&e.LatencyInMs,
			&path,
			&method,
			&customId,
			&e.Request,
			&e.Response,
			&e.UserId,
			&e.Action,
			&e.PolicyId,
			&e.RouteId,
			&e.CorrelationId,
			&e.Metadata,
		); err != nil {
			return nil, err
		}

		pe := &e
		pe.Path = path.String
		pe.Method = method.String
		pe.CustomId = customId.String

		events = append(events, pe)
	}

	return events, nil
}

func (s *Store) GetLatencyPercentiles(start, end int64, tags, keyIds []string) ([]float64, error) {
	eventSelectionBlock := `
	WITH events_table AS
		(
			SELECT * FROM events 
	`

	conditionBlock := fmt.Sprintf("WHERE created_at >= %d AND created_at <= %d ", start, end)
	if len(tags) != 0 {
		conditionBlock += fmt.Sprintf("AND tags @> '%s' ", sliceToSqlStringArray(tags))
	}

	if len(keyIds) != 0 {
		conditionBlock += fmt.Sprintf("AND key_id = ANY('%s')", sliceToSqlStringArray(keyIds))
	}

	if len(tags) != 0 || len(keyIds) != 0 {
		eventSelectionBlock += conditionBlock
	}

	eventSelectionBlock += ")"

	query :=
		`
		SELECT    COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY events_table.latency_in_ms), 0) as median_latency, COALESCE(percentile_cont(0.99) WITHIN GROUP (ORDER BY events_table.latency_in_ms), 0) as top_latency
		FROM      events_table
		`

	query = eventSelectionBlock + query

	ctx, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := []float64{}
	for rows.Next() {
		var median float64
		var top float64

		if err := rows.Scan(
			&median,
			&top,
		); err != nil {
			return nil, err
		}

		data = []float64{
			median,
			top,
		}
	}

	return data, nil
}

func (s *Store) GetCustomIds(keyId string) ([]string, error) {
	query := fmt.Sprintf(`
	SELECT DISTINCT custom_id
	FROM events
	WHERE key_id = '%s' AND custom_id IS NOT NULL AND NOT custom_id = ''
	`, keyId)

	ctx, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []string{}

	for rows.Next() {
		var customId string

		if err := rows.Scan(
			&customId,
		); err != nil {
			return nil, err
		}

		result = append(result, customId)
	}

	return result, nil
}

func (s *Store) GetUserIds(keyId string) ([]string, error) {
	query := fmt.Sprintf(`
	SELECT DISTINCT user_id
	FROM events
	WHERE key_id = '%s' AND NOT user_id = ''
	`, keyId)

	ctx, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []string{}

	for rows.Next() {
		var userId string

		if err := rows.Scan(
			&userId,
		); err != nil {
			return nil, err
		}

		result = append(result, userId)
	}

	return result, nil
}

func (s *Store) GetTopKeyDataPoints(start, end int64, tags, keyIds []string, order string, limit, offset int, name string, revoked *bool) ([]*event.KeyDataPoint, error) {
	args := []any{}
	condition := ""
	condition2 := ""

	index := 1
	if len(tags) > 0 {
		condition += fmt.Sprintf("AND tags @> $%d", index)

		args = append(args, pq.Array(tags))
		index++
	}

	if len(keyIds) > 0 {
		condition += fmt.Sprintf(" AND key_id = ANY($%d)", index)

		args = append(args, pq.Array(keyIds))
		index++
	}

	if len(name) > 0 {
		condition += fmt.Sprintf(" AND LOWER(name) LIKE LOWER('%%%s%%')", name)
	}

	if revoked != nil {
		bools := "False"
		if *revoked {
			bools = "True"
		}

		condition += fmt.Sprintf(" AND revoked = %s", bools)
	}

	if len(tags) > 0 {
		condition2 += fmt.Sprintf("AND keys.tags @> $%d", index)

		args = append(args, pq.Array(tags))
		index++
	}

	if len(keyIds) > 0 {
		condition2 += fmt.Sprintf(" AND keys.key_id = ANY($%d)", index)

		args = append(args, pq.Array(keyIds))
	}

	if len(name) > 0 {
		condition2 += fmt.Sprintf(" AND LOWER(keys.name) LIKE LOWER('%%%s%%')", name)
	}

	if revoked != nil {
		bools := "False"
		if *revoked {
			bools = "True"
		}

		condition2 += fmt.Sprintf(" AND keys.revoked = %s", bools)
	}

	query := fmt.Sprintf(`
	WITH keys_table AS
	(
			SELECT key_id FROM keys WHERE created_at >= %d AND created_at < %d %s
	),top_keys_table AS 
	(
		SELECT 
		events.key_id,
		SUM(cost_in_usd) AS "CostInUsd"
		FROM events
		LEFT JOIN keys
		ON keys.key_id = events.key_id
		WHERE (events.key_id = '') IS FALSE AND events.created_at >= %d AND events.created_at < %d %s
		GROUP BY events.key_id
	)
	SELECT CASE
			WHEN top_keys_table.key_id IS NOT NULL THEN top_keys_table.key_id
			ELSE keys_table.key_id
		END 
		AS key_id
  , COALESCE(top_keys_table."CostInUsd", 0) AS cost_in_usd
		FROM keys_table
		FULL JOIN top_keys_table
		ON top_keys_table.key_id = keys_table.key_id 

`, start, end, condition, start, end, condition2)

	qorder := "DESC"
	if len(order) != 0 && strings.ToUpper(order) == "ASC" {
		qorder = "ASC"
	}

	query += fmt.Sprintf(`
	ORDER BY cost_in_usd %s 
`, qorder)

	if limit != 0 {
		query += fmt.Sprintf(`
		LIMIT %d OFFSET %d;
	`, limit, offset)
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := []*event.KeyDataPoint{}
	for rows.Next() {
		var e event.KeyDataPoint
		var keyId sql.NullString

		additional := []any{
			&keyId,
			&e.CostInUsd,
		}

		if err := rows.Scan(
			additional...,
		); err != nil {
			return nil, err
		}

		pe := &e
		pe.KeyId = keyId.String

		data = append(data, pe)
	}

	return data, nil
}

func (s *Store) GetAggregatedEventByDayDataPoints(start, end int64, keyIds []string) ([]*event.DataPoint, error) {
	conditionBlock := fmt.Sprintf("WHERE time_stamp >= %d AND time_stamp < %d ", start, end)
	if len(keyIds) != 0 {
		conditionBlock += fmt.Sprintf("AND key_id = ANY('%s')", sliceToSqlStringArray(keyIds))
	}

	query := fmt.Sprintf(
		`
		SELECT * FROM event_agg_by_day
		%s
		ORDER BY  event_agg_by_day.time_stamp;
		`,
		conditionBlock,
	)

	ctx, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := []*event.DataPoint{}
	for rows.Next() {
		var e event.DataPoint
		var keyId sql.NullString
		var id sql.NullInt32

		additional := []any{
			&id,
			&e.TimeStamp,
			&e.NumberOfRequests,
			&e.CostInUsd,
			&e.LatencyInMs,
			&e.PromptTokenCount,
			&e.CompletionTokenCount,
			&e.SuccessCount,
			&keyId,
		}

		if err := rows.Scan(
			additional...,
		); err != nil {
			return nil, err
		}

		pe := &e
		pe.KeyId = keyId.String

		data = append(data, pe)
	}

	return data, nil
}

func (s *Store) GetEventDataPoints(start, end, increment int64, tags, keyIds, customIds, userIds []string, filters []string) ([]*event.DataPoint, error) {
	groupByQuery := "GROUP BY time_series_table.series"
	selectQuery := "SELECT series AS time_stamp, COALESCE(COUNT(events_table.event_id),0) AS num_of_requests, COALESCE(SUM(events_table.cost_in_usd),0) AS cost_in_usd, COALESCE(SUM(events_table.latency_in_ms),0) AS latency_in_ms, COALESCE(SUM(events_table.prompt_token_count),0) AS prompt_token_count, COALESCE(SUM(events_table.completion_token_count),0) AS completion_token_count, COALESCE(SUM(CASE WHEN status_code = 200 THEN 1 END),0) AS success_count"

	if len(filters) != 0 {
		for _, filter := range filters {
			if filter == "model" {
				groupByQuery += ",events_table.model"
				selectQuery += ",events_table.model as model"
			}

			if filter == "keyId" {
				groupByQuery += ",events_table.key_id"
				selectQuery += ",events_table.key_id as keyId"
			}

			if filter == "customId" {
				groupByQuery += ",events_table.custom_id"
				selectQuery += ",events_table.custom_id as customId"
			}

			if filter == "userId" {
				groupByQuery += ",events_table.user_id"
				selectQuery += ",events_table.user_id as userId"
			}
		}
	}

	query := fmt.Sprintf(
		`
		,time_series_table AS
		(
			SELECT generate_series(%d, %d, %d) series
		)
		%s
		FROM       time_series_table
		LEFT JOIN  events_table
		ON         events_table.created_at >= time_series_table.series 
		AND        events_table.created_at < time_series_table.series + %d
		%s
		ORDER BY  time_series_table.series;
		`,
		start, end, increment, selectQuery, increment, groupByQuery,
	)

	eventSelectionBlock := `
	WITH events_table AS
		(
			SELECT * FROM events 
	`

	conditionBlock := fmt.Sprintf("WHERE created_at >= %d AND created_at < %d ", start, end)
	if len(tags) != 0 {
		conditionBlock += fmt.Sprintf("AND tags @> '%s' ", sliceToSqlStringArray(tags))
	}

	if len(keyIds) != 0 {
		conditionBlock += fmt.Sprintf("AND key_id = ANY('%s')", sliceToSqlStringArray(keyIds))
	}

	if len(customIds) != 0 {
		conditionBlock += fmt.Sprintf("AND custom_id = ANY('%s')", sliceToSqlStringArray(customIds))
	}

	if len(userIds) != 0 {
		conditionBlock += fmt.Sprintf("AND user_id = ANY('%s')", sliceToSqlStringArray(userIds))
	}

	eventSelectionBlock += conditionBlock
	eventSelectionBlock += ")"

	query = eventSelectionBlock + query

	ctx, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := []*event.DataPoint{}
	for rows.Next() {
		var e event.DataPoint
		var model sql.NullString
		var keyId sql.NullString
		var customId sql.NullString
		var userId sql.NullString

		additional := []any{
			&e.TimeStamp,
			&e.NumberOfRequests,
			&e.CostInUsd,
			&e.LatencyInMs,
			&e.PromptTokenCount,
			&e.CompletionTokenCount,
			&e.SuccessCount,
		}

		if len(filters) != 0 {
			for _, filter := range filters {
				if filter == "model" {
					additional = append(additional, &model)
				}

				if filter == "keyId" {
					additional = append(additional, &keyId)
				}

				if filter == "customId" {
					additional = append(additional, &customId)
				}

				if filter == "userId" {
					additional = append(additional, &userId)
				}
			}
		}

		if err := rows.Scan(
			additional...,
		); err != nil {
			return nil, err
		}

		pe := &e
		pe.Model = model.String
		pe.KeyId = keyId.String
		pe.CustomId = customId.String
		pe.UserId = userId.String

		data = append(data, pe)
	}

	return data, nil
}

func (s *Store) GetEventsV2(req *event.EventRequest) (*event.EventResponse, error) {
	query := fmt.Sprintf(`
		SELECT * FROM events WHERE created_at >= %d AND created_at < %d
	`, req.Start, req.End)

	cquery := fmt.Sprintf(`
	SELECT COUNT(*) FROM events WHERE created_at >= %d AND created_at < %d
`, req.Start, req.End)

	if len(req.UserIds) != 0 {
		query += fmt.Sprintf(" AND user_id = ANY('%s')", sliceToSqlStringArray(req.UserIds))
		cquery += fmt.Sprintf(" AND user_id = ANY('%s')", sliceToSqlStringArray(req.UserIds))
	}

	if req.Status != 0 {
		query += fmt.Sprintf(" AND status_code = %d", req.Status)
		cquery += fmt.Sprintf(" AND status_code = %d", req.Status)
	}

	if len(req.CustomIds) != 0 {
		query += fmt.Sprintf(" AND custom_id = ANY('%s')", sliceToSqlStringArray(req.CustomIds))
		cquery += fmt.Sprintf(" AND custom_id = ANY('%s')", sliceToSqlStringArray(req.CustomIds))
	}

	if len(req.KeyIds) != 0 {
		query += fmt.Sprintf(" AND key_id = ANY('%s')", sliceToSqlStringArray(req.KeyIds))
		cquery += fmt.Sprintf(" AND key_id = ANY('%s')", sliceToSqlStringArray(req.KeyIds))
	}

	if len(req.Tags) != 0 {
		query += fmt.Sprintf(" AND tags @> '%s'", sliceToSqlStringArray(req.Tags))
		cquery += fmt.Sprintf(" AND tags @> '%s'", sliceToSqlStringArray(req.Tags))
	}

	if len(req.PolicyIds) != 0 {
		query += fmt.Sprintf(" AND policy_id = ANY('%s')", sliceToSqlStringArray(req.PolicyIds))
		cquery += fmt.Sprintf(" AND policy_id = ANY('%s')", sliceToSqlStringArray(req.PolicyIds))
	}

	if len(req.Actions) != 0 {
		query += fmt.Sprintf(" AND action = ANY('%s')", sliceToSqlStringArray(req.Actions))
		cquery += fmt.Sprintf(" AND action = ANY('%s')", sliceToSqlStringArray(req.Actions))
	}

	if len(req.CostOrder) != 0 {
		query += fmt.Sprintf(" ORDER BY cost_in_usd %s", strings.ToUpper(req.CostOrder))
	}

	if len(req.DateOrder) != 0 {
		query += fmt.Sprintf(" ORDER BY created_at %s", strings.ToUpper(req.DateOrder))
	}

	if req.Limit != 0 {
		query += fmt.Sprintf(` LIMIT %d OFFSET %d;`, req.Limit, req.Offset)
	}

	qrContext, qrCancel := context.WithTimeout(context.Background(), s.rt)
	defer qrCancel()

	resp := &event.EventResponse{}

	if req.ReturnCount {
		count := 0
		err := s.db.QueryRowContext(qrContext, cquery).Scan(&count)
		if err != nil {
			if err != sql.ErrNoRows {
				return nil, err
			}
		}

		resp.Count = count
	}

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	events := []*event.Event{}
	rows, err := s.db.QueryContext(ctxTimeout, query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var e event.Event
		var path sql.NullString
		var method sql.NullString
		var customId sql.NullString

		if err := rows.Scan(
			&e.Id,
			&e.CreatedAt,
			pq.Array(&e.Tags),
			&e.KeyId,
			&e.CostInUsd,
			&e.Provider,
			&e.Model,
			&e.Status,
			&e.PromptTokenCount,
			&e.CompletionTokenCount,
			&e.LatencyInMs,
			&path,
			&method,
			&customId,
			&e.Request,
			&e.Response,
			&e.UserId,
			&e.Action,
			&e.PolicyId,
			&e.RouteId,
			&e.CorrelationId,
			&e.Metadata,
		); err != nil {
			return nil, err
		}

		pe := &e
		pe.Path = path.String
		pe.Method = method.String
		pe.CustomId = customId.String

		events = append(events, pe)
	}

	resp.Events = events

	return resp, nil
}

func (s *Store) InsertEvent(e *event.Event) error {
	query := `
		INSERT INTO events (event_id, created_at, tags, key_id, cost_in_usd, provider, model, status_code, prompt_token_count, completion_token_count, latency_in_ms, path, method, custom_id, request, response, user_id, action, policy_id, route_id, correlation_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
	`

	values := []any{
		e.Id,
		e.CreatedAt,
		sliceToSqlStringArray(e.Tags),
		e.KeyId,
		e.CostInUsd,
		e.Provider,
		e.Model,
		e.Status,
		e.PromptTokenCount,
		e.CompletionTokenCount,
		e.LatencyInMs,
		e.Path,
		e.Method,
		e.CustomId,
		e.Request,
		e.Response,
		e.UserId,
		e.Action,
		e.PolicyId,
		e.RouteId,
		e.CorrelationId,
		e.Metadata,
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	if _, err := s.db.ExecContext(ctx, query, values...); err != nil {
		return err
	}

	return nil
}
