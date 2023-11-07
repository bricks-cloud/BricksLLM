package postgresql

import (
	"context"
	"database/sql"
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
		ALTER TABLE keys ADD COLUMN IF NOT EXISTS setting_id VARCHAR(255);
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
		INSERT INTO events (event_id, created_at, tags, key_id, cost_in_usd, provider, model, status_code, prompt_token_count, completion_token_count, latency_in_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
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
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	if _, err := s.db.ExecContext(ctx, query, values...); err != nil {
		return err
	}

	return nil
}

func (s *Store) GetLatencyPercentiles(start, end int64, tags, keyIds []string) ([]float64, error) {
	eventSelectionBlock := `
	WITH events_table AS
		(
			SELECT * FROM events 
	`

	conditionBlock := fmt.Sprintf("WHERE created_at >= %d AND created_at <= %d", start, end)
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

func (s *Store) GetEventDataPoints(start, end, increment int64, tags, keyIds []string) ([]*event.DataPoint, error) {
	query := fmt.Sprintf(
		`
		,time_series_table AS
		(
			SELECT generate_series(%d, %d, %d) series
		)
		SELECT    series AS time_stamp, COALESCE(COUNT(events_table.event_id),0) AS num_of_requests, COALESCE(SUM(events_table.cost_in_usd),0) AS cost_in_usd, COALESCE(SUM(events_table.latency_in_ms),0) AS latency_in_ms, COALESCE(SUM(events_table.prompt_token_count),0) AS prompt_token_count, COALESCE(SUM(events_table.completion_token_count),0) AS completion_token_count, COALESCE(SUM(CASE WHEN status_code = 200 THEN 1 END),0) AS success_count
		FROM      time_series_table
		LEFT JOIN events_table
		ON        events_table.created_at >= time_series_table.series 
		AND       events_table.created_at < time_series_table.series + %d
		GROUP BY  time_series_table.series
		ORDER BY  time_series_table.series;
		`,
		start, end, increment, increment,
	)

	eventSelectionBlock := `
	WITH events_table AS
		(
			SELECT * FROM events 
	`

	conditionBlock := "WHERE "
	if len(tags) != 0 {
		conditionBlock += fmt.Sprintf("tags @> '%s' ", sliceToSqlStringArray(tags))
	}

	if len(tags) != 0 && len(keyIds) != 0 {
		conditionBlock += "AND "
	}

	if len(keyIds) != 0 {
		conditionBlock += fmt.Sprintf("key_id = ANY('%s')", sliceToSqlStringArray(keyIds))
	}

	if len(tags) != 0 || len(keyIds) != 0 {
		eventSelectionBlock += conditionBlock
	}

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
		if err := rows.Scan(
			&e.TimeStamp,
			&e.NumberOfRequests,
			&e.CostInUsd,
			&e.LatencyInMs,
			&e.PromptTokenCount,
			&e.CompletionTokenCount,
			&e.SuccessCouunt,
		); err != nil {
			return nil, err
		}

		data = append(data, &e)
	}

	return data, nil
}

func (s *Store) GetKeysByTag(tag string) ([]*key.ResponseKey, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctxTimeout, "SELECT * FROM keys WHERE $1 = ANY(tags)", tag)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := []*key.ResponseKey{}
	for rows.Next() {
		var k key.ResponseKey
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
			&k.SettingId,
		); err != nil {
			return nil, err
		}
		keys = append(keys, &k)
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
			&k.SettingId,
		); err != nil {
			return nil, err
		}
		keys = append(keys, &k)
	}

	if len(keys) == 0 {
		return nil, nil
	}

	return keys[0], nil
}

func (s *Store) GetProviderSetting(id string) (*provider.Setting, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctxTimeout, "SELECT * FROM provider_settings WHERE $1 = id", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := []*provider.Setting{}
	for rows.Next() {
		setting := &provider.Setting{}
		var data []byte
		if err := rows.Scan(
			&setting.Id,
			&setting.CreatedAt,
			&setting.UpdatedAt,
			&setting.Provider,
			&data,
		); err != nil {
			return nil, err
		}

		m := map[string]string{}
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, err
		}

		setting.Setting = m
		settings = append(settings, setting)
	}

	if len(settings) == 1 {
		return settings[0], nil
	}

	return nil, internal_errors.NewNotFoundError("provider setting is not found")
}

func (s *Store) GetAllProviderSettings() ([]*provider.Setting, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctxTimeout, "SELECT * FROM provider_settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := []*provider.Setting{}
	for rows.Next() {
		setting := &provider.Setting{}
		var data []byte
		if err := rows.Scan(
			&setting.Id,
			&setting.CreatedAt,
			&setting.UpdatedAt,
			&setting.Provider,
			&data,
		); err != nil {
			return nil, err
		}

		m := map[string]string{}
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, err
		}

		setting.Setting = m
		settings = append(settings, setting)
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
			&k.SettingId,
		); err != nil {
			return nil, err
		}
		keys = append(keys, &k)
	}

	return keys, nil
}

func (s *Store) GetUpdatedProviderSettings(interval time.Duration) ([]*provider.Setting, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctxTimeout, "SELECT * FROM provider_settings WHERE updated_at >= $1", time.Now().Unix()-int64(interval.Seconds()))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := []*provider.Setting{}
	for rows.Next() {
		setting := &provider.Setting{}
		var data []byte
		if err := rows.Scan(
			&setting.Id,
			&setting.CreatedAt,
			&setting.UpdatedAt,
			&setting.Provider,
			&data,
		); err != nil {
			return nil, err
		}

		m := map[string]string{}
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, err
		}

		setting.Setting = m
		settings = append(settings, setting)
	}

	return settings, nil
}

func (s *Store) GetUpdatedKeys(interval time.Duration) ([]*key.ResponseKey, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctxTimeout, "SELECT * FROM keys WHERE updated_at >= $1", time.Now().Unix()-int64(interval.Seconds()))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := []*key.ResponseKey{}
	for rows.Next() {
		var k key.ResponseKey
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
			&k.SettingId,
		); err != nil {
			return nil, err
		}
		keys = append(keys, &k)
	}

	return keys, nil
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
		values = append(values, uk.Revoked)
		fields = append(fields, fmt.Sprintf("revoked = $%d", counter))
		counter++
	}

	if len(uk.RevokedReason) != 0 {
		values = append(values, uk.RevokedReason)
		fields = append(fields, fmt.Sprintf("revoked_reason = $%d", counter))
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

	if len(uk.Ttl) != 0 {
		values = append(values, uk.Ttl)
		fields = append(fields, fmt.Sprintf("ttl = $%d", counter))
		counter++
	}

	if len(uk.SettingId) != 0 {
		values = append(values, uk.SettingId)
		fields = append(fields, fmt.Sprintf("setting_id = $%d", counter))
		counter++
	}

	query := fmt.Sprintf("UPDATE keys SET %s WHERE key_id = $1 RETURNING *;", strings.Join(fields, ","))

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	var k key.ResponseKey
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
		&k.SettingId,
	); err != nil {
		return nil, err
	}

	return &k, nil
}

func (s *Store) UpdateProviderSetting(id string, setting *provider.Setting) (*provider.Setting, error) {
	data, err := json.Marshal(setting.Setting)
	if err != nil {
		return nil, err
	}

	values := []any{
		id,
		data,
		setting.UpdatedAt,
	}

	query := "UPDATE provider_settings SET setting = $2, updated_at = $3 WHERE id = $1 RETURNING id, created_at, updated_at, provider;"
	updated := &provider.Setting{}
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	if err := s.db.QueryRowContext(ctxTimeout, query, values...).Scan(
		&updated.Id,
		&updated.CreatedAt,
		&updated.UpdatedAt,
		&updated.Provider,
	); err != nil {
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
		INSERT INTO provider_settings (id, created_at, updated_at, provider, setting)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at, provider
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
	}

	created := &provider.Setting{}
	if err := s.db.QueryRowContext(ctxTimeout, query, values...).Scan(
		&created.Id,
		&created.CreatedAt,
		&created.UpdatedAt,
		&created.Provider,
	); err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Store) CreateKey(rk *key.RequestKey) (*key.ResponseKey, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	duplicated, err := s.db.QueryContext(ctxTimeout, "SELECT * FROM keys WHERE $1 = key", rk.Key)
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
		INSERT INTO keys (name, created_at, updated_at, tags, revoked, key_id, key, revoked_reason, cost_limit_in_usd, cost_limit_in_usd_over_time, cost_limit_in_usd_unit, rate_limit_over_time, rate_limit_unit, ttl, setting_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING *;
	`

	values := []any{
		rk.Name,
		rk.CreatedAt,
		rk.UpdatedAt,
		sliceToSqlStringArray(rk.Tags),
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
	}

	ctxTimeout, cancel = context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	var k key.ResponseKey
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
		&k.SettingId,
	); err != nil {
		return nil, err
	}

	return &k, nil
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
