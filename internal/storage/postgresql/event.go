package postgresql

import (
	"context"
	"database/sql"
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

func (s *Store) GetTopKeyDataPoints(start, end int64, tags, keyIds []string, order string, limit, offset int) ([]*event.KeyDataPoint, error) {
	args := []any{}
	condition := ""
	index := 1
	if len(tags) > 0 {
		condition = fmt.Sprintf("AND tags @> $%d", index)
		args = append(args, pq.Array(tags))
		index++
	}

	if len(keyIds) > 0 {
		condition += fmt.Sprintf(" AND key_id = ANY($%d)", index)
		args = append(args, pq.Array(keyIds))
	}

	query := fmt.Sprintf(`
	WITH keys_table AS
	(
			SELECT key_id FROM keys WHERE created_at >= %d AND created_at < %d
	),top_keys_table AS 
	(
		SELECT 
		key_id,
		SUM(cost_in_usd) AS "CostInUsd"
	FROM events
	WHERE (key_id = '') IS FALSE AND created_at >= %d AND created_at < %d %s
	GROUP BY key_id
	)
	SELECT keys_table.key_id, COALESCE(top_keys_table."CostInUsd", 0) AS cost_in_usd
	FROM keys_table
	LEFT JOIN top_keys_table
	ON top_keys_table.key_id = keys_table.key_id
`, start, end, start, end, condition)

	qorder := "DESC"
	if len(order) != 0 && strings.ToUpper(order) == "ASC" {
		qorder = "ASC"
	}

	if limit != 0 {
		query += fmt.Sprintf(`
		ORDER BY cost_in_usd %s
		LIMIT %d OFFSET %d;
	`, qorder, limit, offset)
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	for _, t := range args {
		fmt.Println(t)
	}

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
