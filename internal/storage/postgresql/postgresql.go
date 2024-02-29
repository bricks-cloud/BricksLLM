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
	"github.com/bricks-cloud/bricksllm/internal/key"
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

func (s *Store) CreateKeysTable() error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS keys (
		name VARCHAR(255) NOT NULL,
		created_at BIGINT NOT NULL,
		updated_at BIGINT NOT NULL,
		tags VARCHAR(255)[],
		revoked BOOLEAN NOT NULL,
		key_id VARCHAR(255) PRIMARY KEY,
		key VARCHAR(255) NOT NULL,
		revoked_reason VARCHAR(255),
		cost_limit_in_usd FLOAT8,
		cost_limit_in_usd_over_time FLOAT8,
		cost_limit_in_usd_unit VARCHAR(255),
		rate_limit_over_time INT,
		rate_limit_unit VARCHAR(255),
		ttl VARCHAR(255)
	)`

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	_, err := s.db.ExecContext(ctxTimeout, createTableQuery)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) AlterKeysTable() error {
	alterTableQuery := `
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 
				FROM pg_constraint 
				WHERE conname = 'key_uniqueness'
			) THEN
				ALTER TABLE keys
				ADD CONSTRAINT key_uniqueness UNIQUE (key);
			END IF;
		END
		$$;
		ALTER TABLE keys ADD COLUMN IF NOT EXISTS setting_id VARCHAR(255), ADD COLUMN IF NOT EXISTS allowed_paths JSONB, ADD COLUMN IF NOT EXISTS setting_ids VARCHAR(255)[] NOT NULL DEFAULT ARRAY[]::VARCHAR(255)[], ADD COLUMN IF NOT EXISTS should_log_request BOOLEAN NOT NULL DEFAULT FALSE, ADD COLUMN IF NOT EXISTS should_log_response BOOLEAN NOT NULL DEFAULT FALSE, ADD COLUMN IF NOT EXISTS rotation_enabled BOOLEAN NOT NULL DEFAULT FALSE;
	`

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	_, err := s.db.ExecContext(ctxTimeout, alterTableQuery)
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

func (s *Store) DropProviderSettingsTable() error {
	dropTableQuery := `DROP TABLE provider_settings`
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	_, err := s.db.ExecContext(ctxTimeout, dropTableQuery)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) DropEventsTable() error {
	dropTableQuery := `DROP TABLE events`
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	_, err := s.db.ExecContext(ctxTimeout, dropTableQuery)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) DropKeysTable() error {
	dropTableQuery := `DROP TABLE keys`
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	_, err := s.db.ExecContext(ctxTimeout, dropTableQuery)
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

func shouldAddAnd(userId, customId string, keyIds []string) bool {
	num := 0

	if len(userId) == 0 {
		num++
	}

	if len(customId) == 0 {
		num++
	}

	if len(keyIds) == 0 {
		num++
	}

	return num >= 2
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
		break
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

	conditionBlock := fmt.Sprintf("WHERE created_at >= %d AND created_at <= %d ", start, end)
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

func (s *Store) GetKeys(tags, keyIds []string, provider string) ([]*key.ResponseKey, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	args := []any{}

	query := ""

	selectionQuery := "SELECT * FROM keys "

	index := 1
	if len(tags) != 0 {
		args = append(args, pq.Array(tags))
		index += 1
		selectionQuery += "WHERE tags @> $1"
	}

	if len(keyIds) != 0 {
		args = append(args, pq.Array(keyIds))

		if index != 1 {
			selectionQuery += fmt.Sprintf(" AND key_id = ANY($%d)", index)
		}

		if index == 1 {
			selectionQuery += fmt.Sprintf("WHERE key_id = ANY($%d)", index)
		}

		index += 1
	}

	query = selectionQuery

	if len(provider) != 0 {
		args = append(args, provider)
		query = fmt.Sprintf(`
			WITH keys_table AS
			(
				%s
			),provider_settings_table AS
			(
				SELECT * FROM provider_settings WHERE $%d = provider
			)
			SELECT DISTINCT keys_table.*
			FROM keys_table
			JOIN provider_settings_table
			ON keys_table.setting_id = provider_settings_table.id
			OR provider_settings_table.id = ANY(keys_table.setting_ids);
		`, selectionQuery, index)
	}

	rows, err := s.db.QueryContext(ctxTimeout, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, internal_errors.NewNotFoundError("keys are not found")
		}

		return nil, err
	}
	defer rows.Close()

	keys := []*key.ResponseKey{}
	for rows.Next() {
		var k key.ResponseKey
		var settingId sql.NullString
		var data []byte
		if err := rows.Scan(
			&k.Name,
			&k.CreatedAt,
			&k.UpdatedAt,
			pq.Array(&k.Tags),
			&k.Revoked,
			&k.KeyId,
			&k.Key,
			&k.RevokedReason,
			&k.CostLimitInUsd,
			&k.CostLimitInUsdOverTime,
			&k.CostLimitInUsdUnit,
			&k.RateLimitOverTime,
			&k.RateLimitUnit,
			&k.Ttl,
			&settingId,
			&data,
			pq.Array(&k.SettingIds),
			&k.ShouldLogRequest,
			&k.ShouldLogResponse,
			&k.RotationEnabled,
		); err != nil {
			return nil, err
		}

		pk := &k
		pk.SettingId = settingId.String

		if len(data) != 0 {
			pathConfigs := []key.PathConfig{}
			if err := json.Unmarshal(data, &pathConfigs); err != nil {
				return nil, err
			}

			pk.AllowedPaths = pathConfigs
		}

		keys = append(keys, pk)
	}

	return keys, nil
}

func (s *Store) GetKey(keyId string) (*key.ResponseKey, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctxTimeout, "SELECT * FROM keys WHERE key_id = $1", keyId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := []*key.ResponseKey{}
	for rows.Next() {
		var k key.ResponseKey
		var settingId sql.NullString
		var data []byte

		if err := rows.Scan(
			&k.Name,
			&k.CreatedAt,
			&k.UpdatedAt,
			pq.Array(&k.Tags),
			&k.Revoked,
			&k.KeyId,
			&k.Key,
			&k.RevokedReason,
			&k.CostLimitInUsd,
			&k.CostLimitInUsdOverTime,
			&k.CostLimitInUsdUnit,
			&k.RateLimitOverTime,
			&k.RateLimitUnit,
			&k.Ttl,
			&settingId,
			&data,
			pq.Array(&k.SettingIds),
			&k.ShouldLogRequest,
			&k.ShouldLogResponse,
			&k.RotationEnabled,
		); err != nil {
			return nil, err
		}

		pk := &k
		pk.SettingId = settingId.String

		if len(data) != 0 {
			pathConfigs := []key.PathConfig{}
			if err := json.Unmarshal(data, &pathConfigs); err != nil {
				return nil, err
			}

			pk.AllowedPaths = pathConfigs
		}

		keys = append(keys, pk)
	}

	if len(keys) == 0 {
		return nil, nil
	}

	return keys[0], nil
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

func (s *Store) GetAllKeys() ([]*key.ResponseKey, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctxTimeout, "SELECT * FROM keys")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := []*key.ResponseKey{}
	for rows.Next() {
		var k key.ResponseKey
		var settingId sql.NullString
		var data []byte
		if err := rows.Scan(
			&k.Name,
			&k.CreatedAt,
			&k.UpdatedAt,
			pq.Array(&k.Tags),
			&k.Revoked,
			&k.KeyId,
			&k.Key,
			&k.RevokedReason,
			&k.CostLimitInUsd,
			&k.CostLimitInUsdOverTime,
			&k.CostLimitInUsdUnit,
			&k.RateLimitOverTime,
			&k.RateLimitUnit,
			&k.Ttl,
			&settingId,
			&data,
			pq.Array(&k.SettingIds),
			&k.ShouldLogRequest,
			&k.ShouldLogResponse,
			&k.RotationEnabled,
		); err != nil {
			return nil, err
		}
		pk := &k
		pk.SettingId = settingId.String

		if len(data) != 0 {
			pathConfigs := []key.PathConfig{}
			if err := json.Unmarshal(data, &pathConfigs); err != nil {
				return nil, err
			}

			pk.AllowedPaths = pathConfigs
		}

		keys = append(keys, pk)
	}

	return keys, nil
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

func (s *Store) GetUpdatedKeys(updatedAt int64) ([]*key.ResponseKey, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctxTimeout, "SELECT * FROM keys WHERE updated_at >= $1", updatedAt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := []*key.ResponseKey{}
	for rows.Next() {
		var k key.ResponseKey
		var settingId sql.NullString
		var data []byte
		if err := rows.Scan(
			&k.Name,
			&k.CreatedAt,
			&k.UpdatedAt,
			pq.Array(&k.Tags),
			&k.Revoked,
			&k.KeyId,
			&k.Key,
			&k.RevokedReason,
			&k.CostLimitInUsd,
			&k.CostLimitInUsdOverTime,
			&k.CostLimitInUsdUnit,
			&k.RateLimitOverTime,
			&k.RateLimitUnit,
			&k.Ttl,
			&settingId,
			&data,
			pq.Array(&k.SettingIds),
			&k.ShouldLogRequest,
			&k.ShouldLogResponse,
			&k.RotationEnabled,
		); err != nil {
			return nil, err
		}

		pk := &k
		pk.SettingId = settingId.String
		if len(data) != 0 {
			pathConfigs := []key.PathConfig{}
			if err := json.Unmarshal(data, &pathConfigs); err != nil {
				return nil, err
			}

			pk.AllowedPaths = pathConfigs
		}

		keys = append(keys, pk)
	}

	return keys, nil
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

func (s *Store) UpdateKey(id string, uk *key.UpdateKey) (*key.ResponseKey, error) {
	fields := []string{}
	counter := 2
	values := []any{
		id,
	}

	if len(uk.Name) != 0 {
		values = append(values, uk.Name)
		fields = append(fields, fmt.Sprintf("name = $%d", counter))
		counter++
	}

	if uk.UpdatedAt != 0 {
		values = append(values, uk.UpdatedAt)
		fields = append(fields, fmt.Sprintf("updated_at = $%d", counter))
		counter++
	}

	if len(uk.Tags) != 0 {
		values = append(values, sliceToSqlStringArray(uk.Tags))
		fields = append(fields, fmt.Sprintf("tags = $%d", counter))
		counter++
	}

	if uk.Revoked != nil {
		if *uk.Revoked && len(uk.RevokedReason) != 0 {
			values = append(values, uk.RevokedReason)
			fields = append(fields, fmt.Sprintf("revoked_reason = $%d", counter))
			counter++
		}

		if !*uk.Revoked {
			values = append(values, "")
			fields = append(fields, fmt.Sprintf("revoked_reason = $%d", counter))
			counter++
		}

		values = append(values, uk.Revoked)
		fields = append(fields, fmt.Sprintf("revoked = $%d", counter))
		counter++
	}

	if uk.CostLimitInUsd != 0 {
		values = append(values, uk.CostLimitInUsd)
		fields = append(fields, fmt.Sprintf("cost_limit_in_usd = $%d", counter))
		counter++
	}

	if uk.CostLimitInUsdOverTime != 0 {
		values = append(values, uk.CostLimitInUsdOverTime)
		fields = append(fields, fmt.Sprintf("cost_limit_in_usd_over_time = $%d", counter))
		counter++
	}

	if len(uk.CostLimitInUsdUnit) != 0 {
		values = append(values, uk.CostLimitInUsdUnit)
		fields = append(fields, fmt.Sprintf("cost_limit_in_usd_unit = $%d", counter))
		counter++
	}

	if uk.RateLimitOverTime != 0 {
		values = append(values, uk.RateLimitOverTime)
		fields = append(fields, fmt.Sprintf("rate_limit_over_time = $%d", counter))
		counter++
	}

	if len(uk.RateLimitUnit) != 0 {
		values = append(values, uk.RateLimitUnit)
		fields = append(fields, fmt.Sprintf("rate_limit_unit = $%d", counter))
		counter++
	}

	if len(uk.SettingId) != 0 {
		values = append(values, uk.SettingId)
		fields = append(fields, fmt.Sprintf("setting_id = $%d", counter))
		counter++
	}

	if len(uk.SettingIds) != 0 {
		values = append(values, sliceToSqlStringArray(uk.SettingIds))
		fields = append(fields, fmt.Sprintf("setting_ids = $%d", counter))
		counter++
	}

	if uk.ShouldLogRequest != nil {
		values = append(values, *uk.ShouldLogRequest)
		fields = append(fields, fmt.Sprintf("should_log_request = $%d", counter))
		counter++
	}

	if uk.ShouldLogResponse != nil {
		values = append(values, *uk.ShouldLogResponse)
		fields = append(fields, fmt.Sprintf("should_log_response = $%d", counter))
		counter++
	}

	if uk.RotationEnabled != nil {
		values = append(values, *uk.RotationEnabled)
		fields = append(fields, fmt.Sprintf("rotation_enabled = $%d", counter))
		counter++
	}

	if uk.AllowedPaths != nil {
		data, err := json.Marshal(uk.AllowedPaths)
		if err != nil {
			return nil, err
		}

		values = append(values, data)
		fields = append(fields, fmt.Sprintf("allowed_paths = $%d", counter))
	}

	query := fmt.Sprintf("UPDATE keys SET %s WHERE key_id = $1 RETURNING *;", strings.Join(fields, ","))

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	var k key.ResponseKey
	var settingId sql.NullString
	var data []byte
	if err := s.db.QueryRowContext(ctxTimeout, query, values...).Scan(
		&k.Name,
		&k.CreatedAt,
		&k.UpdatedAt,
		pq.Array(&k.Tags),
		&k.Revoked,
		&k.KeyId,
		&k.Key,
		&k.RevokedReason,
		&k.CostLimitInUsd,
		&k.CostLimitInUsdOverTime,
		&k.CostLimitInUsdUnit,
		&k.RateLimitOverTime,
		&k.RateLimitUnit,
		&k.Ttl,
		&settingId,
		&data,
		pq.Array(&k.SettingIds),
		&k.ShouldLogRequest,
		&k.ShouldLogResponse,
		&k.RotationEnabled,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, internal_errors.NewNotFoundError(fmt.Sprintf("key not found for id: %s", id))
		}
		return nil, err
	}

	pk := &k
	pk.SettingId = settingId.String

	if len(data) != 0 {
		pathConfigs := []key.PathConfig{}
		if err := json.Unmarshal(data, &pathConfigs); err != nil {
			return nil, err
		}

		pk.AllowedPaths = pathConfigs
	}

	return pk, nil
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

func (s *Store) CreateKey(rk *key.RequestKey) (*key.ResponseKey, error) {
	query := `
		INSERT INTO keys (name, created_at, updated_at, tags, revoked, key_id, key, revoked_reason, cost_limit_in_usd, cost_limit_in_usd_over_time, cost_limit_in_usd_unit, rate_limit_over_time, rate_limit_unit, ttl, setting_id, allowed_paths, setting_ids, should_log_request, should_log_response, rotation_enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		RETURNING *;
	`

	rdata, err := json.Marshal(rk.AllowedPaths)
	if err != nil {
		return nil, err
	}

	values := []any{
		rk.Name,
		rk.CreatedAt,
		rk.UpdatedAt,
		pq.Array(rk.Tags),
		false,
		rk.KeyId,
		rk.Key,
		"",
		rk.CostLimitInUsd,
		rk.CostLimitInUsdOverTime,
		rk.CostLimitInUsdUnit,
		rk.RateLimitOverTime,
		rk.RateLimitUnit,
		rk.Ttl,
		rk.SettingId,
		rdata,
		sliceToSqlStringArray(rk.SettingIds),
		rk.ShouldLogRequest,
		rk.ShouldLogResponse,
		rk.RotationEnabled,
	}

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	var k key.ResponseKey

	var settingId sql.NullString
	var data []byte
	if err := s.db.QueryRowContext(ctxTimeout, query, values...).Scan(
		&k.Name,
		&k.CreatedAt,
		&k.UpdatedAt,
		pq.Array(&k.Tags),
		&k.Revoked,
		&k.KeyId,
		&k.Key,
		&k.RevokedReason,
		&k.CostLimitInUsd,
		&k.CostLimitInUsdOverTime,
		&k.CostLimitInUsdUnit,
		&k.RateLimitOverTime,
		&k.RateLimitUnit,
		&k.Ttl,
		&settingId,
		&data,
		pq.Array(&k.SettingIds),
		&k.ShouldLogRequest,
		&k.ShouldLogResponse,
		&k.RotationEnabled,
	); err != nil {
		return nil, err
	}

	pk := &k
	pk.SettingId = settingId.String

	if len(data) != 0 {
		pathConfigs := []key.PathConfig{}
		if err := json.Unmarshal(data, &pathConfigs); err != nil {
			return nil, err
		}

		pk.AllowedPaths = pathConfigs
	}

	return pk, nil
}

func (s *Store) DeleteKey(id string) error {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	_, err := s.db.ExecContext(ctxTimeout, "DELETE FROM keys WHERE key_id = $1", id)
	return err
}

func sliceToSqlStringArray(slice []string) string {
	return "{" + strings.Join(slice, ",") + "}"
}
