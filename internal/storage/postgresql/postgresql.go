package postgresql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
	"github.com/bricks-cloud/bricksllm/internal/event"
	"github.com/bricks-cloud/bricksllm/internal/provider"

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

func (s *Store) CreateProviderSettingsTable() error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS provider_settings (
		id VARCHAR(255) PRIMARY KEY,
		created_at BIGINT NOT NULL,
		updated_at BIGINT NOT NULL,
		provider VARCHAR(255) NOT NULL,
		setting JSONB NOT NULL
	)`

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	_, err := s.db.ExecContext(ctxTimeout, createTableQuery)
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

func (s *Store) AlterProviderSettingsTable() error {
	alterTableQuery := `
		ALTER TABLE provider_settings ADD COLUMN IF NOT EXISTS name VARCHAR(255), ADD COLUMN IF NOT EXISTS allowed_models VARCHAR(255)[]
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

func (s *Store) GetProviderSetting(id string) (*provider.Setting, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	setting := &provider.Setting{}
	var data []byte
	var name sql.NullString
	err := s.db.QueryRowContext(ctxTimeout, "SELECT * FROM provider_settings WHERE $1 = id", id).Scan(
		&setting.Id,
		&setting.CreatedAt,
		&setting.UpdatedAt,
		&setting.Provider,
		&data,
		&name,
		pq.Array(&setting.AllowedModels),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, internal_errors.NewNotFoundError("provider setting is not found")
		}

		return nil, err
	}

	return setting, nil
}

func (s *Store) GetProviderSettings(withSecret bool, ids []string) ([]*provider.Setting, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	values := []any{}

	query := "SELECT * FROM provider_settings"

	if len(ids) != 0 {
		query += " WHERE id = ANY($1)"
		values = append(values, pq.Array(ids))
	}

	rows, err := s.db.QueryContext(ctxTimeout, query, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := []*provider.Setting{}
	for rows.Next() {
		setting := &provider.Setting{}
		var data []byte
		var name sql.NullString
		if err := rows.Scan(
			&setting.Id,
			&setting.CreatedAt,
			&setting.UpdatedAt,
			&setting.Provider,
			&data,
			&name,
			pq.Array(&setting.AllowedModels),
		); err != nil {
			return nil, err
		}

		if withSecret {
			m := map[string]string{}
			if err := json.Unmarshal(data, &m); err != nil {
				return nil, err
			}
			setting.Setting = m
		}

		setting.Name = name.String
		settings = append(settings, setting)
	}

	if len(ids) != 0 && len(ids) != len(settings) {
		return nil, errors.New("not all settings are found")
	}

	return settings, nil
}

func (s *Store) GetUpdatedProviderSettings(updatedAt int64) ([]*provider.Setting, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctxTimeout, "SELECT * FROM provider_settings WHERE updated_at >= $1", updatedAt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := []*provider.Setting{}
	for rows.Next() {
		setting := &provider.Setting{}
		var data []byte
		var name sql.NullString
		if err := rows.Scan(
			&setting.Id,
			&setting.CreatedAt,
			&setting.UpdatedAt,
			&setting.Provider,
			&data,
			&name,
			pq.Array(&setting.AllowedModels),
		); err != nil {
			return nil, err
		}

		m := map[string]string{}
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, err
		}

		setting.Setting = m
		setting.Name = name.String
		settings = append(settings, setting)
	}

	return settings, nil
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

func (s *Store) UpdateProviderSetting(id string, setting *provider.UpdateSetting) (*provider.Setting, error) {
	values := []any{
		id,
		setting.UpdatedAt,
	}
	fields := []string{"updated_at = $2"}

	d := 3

	if len(setting.Setting) != 0 {
		data, err := json.Marshal(setting.Setting)
		if err != nil {
			return nil, err
		}

		values = append(values, data)
		fields = append(fields, fmt.Sprintf("setting = $%d", d))
		d++
	}

	if setting.Name != nil {
		values = append(values, *setting.Name)
		fields = append(fields, fmt.Sprintf("name = $%d", d))
		d++
	}

	if setting.AllowedModels != nil {
		values = append(values, sliceToSqlStringArray(*setting.AllowedModels))
		fields = append(fields, fmt.Sprintf("allowed_models = $%d", d))
	}

	query := fmt.Sprintf("UPDATE provider_settings SET %s WHERE id = $1 RETURNING id, created_at, updated_at, provider, name, allowed_models;", strings.Join(fields, ","))
	updated := &provider.Setting{}
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	row := s.db.QueryRowContext(ctxTimeout, query, values...)
	if err := row.Scan(
		&updated.Id,
		&updated.CreatedAt,
		&updated.UpdatedAt,
		&updated.Provider,
		&updated.Name,
		pq.Array(&updated.AllowedModels),
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, internal_errors.NewNotFoundError("provider setting is not found for: " + id)
		}

		return nil, err
	}

	return updated, nil
}

func (s *Store) CreateProviderSetting(setting *provider.Setting) (*provider.Setting, error) {
	if len(setting.Provider) == 0 {
		return nil, errors.New("provider is empty")
	}

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	duplicated, err := s.db.QueryContext(ctxTimeout, "SELECT * FROM provider_settings WHERE $1 = id", setting.Id)
	if err != nil {
		return nil, err
	}
	defer duplicated.Close()

	i := 0
	for duplicated.Next() {
		i++
	}

	if i > 0 {
		return nil, NewDuplicationError("key can not be duplicated")
	}

	query := `
		INSERT INTO provider_settings (id, created_at, updated_at, provider, setting, name, allowed_models)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at, provider, name, allowed_models
	`

	data, err := json.Marshal(setting.Setting)
	if err != nil {
		return nil, err
	}

	values := []any{
		setting.Id,
		setting.CreatedAt,
		setting.UpdatedAt,
		setting.Provider,
		data,
		setting.Name,
		sliceToSqlStringArray(setting.AllowedModels),
	}

	created := &provider.Setting{}
	var name sql.NullString
	if err := s.db.QueryRowContext(ctxTimeout, query, values...).Scan(
		&created.Id,
		&created.CreatedAt,
		&created.UpdatedAt,
		&created.Provider,
		&name,
		pq.Array(&created.AllowedModels),
	); err != nil {
		return nil, err
	}

	created.Name = name.String
	return created, nil
}
