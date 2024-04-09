package postgresql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/event"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type Store struct {
	db *sql.DB
	wt time.Duration
	rt time.Duration
}

func NewStore(connStr string, wt time.Duration, rt time.Duration) (*Store, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	return &Store{
		db: db,
		wt: wt,
		rt: rt,
	}, nil
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

func (s *Store) AlterEventsTable() error {
	alterTableQuery := `
		ALTER TABLE events ADD COLUMN IF NOT EXISTS path VARCHAR(255), ADD COLUMN IF NOT EXISTS method VARCHAR(255), ADD COLUMN IF NOT EXISTS custom_id VARCHAR(255), ADD COLUMN IF NOT EXISTS request JSONB, ADD COLUMN IF NOT EXISTS response JSONB, ADD COLUMN IF NOT EXISTS user_id VARCHAR(255) NOT NULL DEFAULT '';
	`

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	_, err := s.db.ExecContext(ctxTimeout, alterTableQuery)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) InsertEvent(e *event.Event) error {
	query := `
		INSERT INTO events (event_id, created_at, tags, key_id, cost_in_usd, provider, model, status_code, prompt_token_count, completion_token_count, latency_in_ms, path, method, custom_id, request, response, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
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
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	if _, err := s.db.ExecContext(ctx, query, values...); err != nil {
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

type NullArray struct {
	Array []string
	Valid bool
}

func (na *NullArray) Scan(value any) error {
	if value == nil {
		na.Array, na.Valid = []string{}, false
		return nil
	}

	na.Valid = true
	return convertAssign(&na.Array, value)
}

func convertAssign(dest, src any) error {
	switch s := src.(type) {
	case []string:
		switch d := dest.(type) {
		case *[]string:
			if d == nil {
				return errors.New("source is nil")
			}

			*d = s
			return nil
		}
	}

	return nil
}

// Value implements the driver Valuer interface.
func (na NullArray) Value() (driver.Value, error) {
	if !na.Valid {
		return nil, nil
	}

	return na.Array, nil
}
