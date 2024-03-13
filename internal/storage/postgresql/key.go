package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/lib/pq"
)

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
		ALTER TABLE keys ADD COLUMN IF NOT EXISTS setting_id VARCHAR(255), ADD COLUMN IF NOT EXISTS allowed_paths JSONB, ADD COLUMN IF NOT EXISTS setting_ids VARCHAR(255)[] NOT NULL DEFAULT ARRAY[]::VARCHAR(255)[], ADD COLUMN IF NOT EXISTS should_log_request BOOLEAN NOT NULL DEFAULT FALSE, ADD COLUMN IF NOT EXISTS should_log_response BOOLEAN NOT NULL DEFAULT FALSE, ADD COLUMN IF NOT EXISTS rotation_enabled BOOLEAN NOT NULL DEFAULT FALSE, ADD COLUMN IF NOT EXISTS policy_ids VARCHAR(255)[] NOT NULL DEFAULT ARRAY[]::VARCHAR(255)[];
	`

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	_, err := s.db.ExecContext(ctxTimeout, alterTableQuery)
	if err != nil {
		return err
	}

	return nil
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
			pq.Array(&k.PolicyIds),
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
			pq.Array(&k.PolicyIds),
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
			pq.Array(&k.PolicyIds),
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
			pq.Array(&k.PolicyIds),
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

	if uk.CostLimitInUsd != nil {
		values = append(values, *uk.CostLimitInUsd)
		fields = append(fields, fmt.Sprintf("cost_limit_in_usd = $%d", counter))
		counter++
	}

	if uk.CostLimitInUsdOverTime != nil {
		values = append(values, *uk.CostLimitInUsdOverTime)
		fields = append(fields, fmt.Sprintf("cost_limit_in_usd_over_time = $%d", counter))
		counter++
	}

	if uk.CostLimitInUsdUnit != nil {
		values = append(values, *uk.CostLimitInUsdUnit)
		fields = append(fields, fmt.Sprintf("cost_limit_in_usd_unit = $%d", counter))
		counter++
	}

	if uk.RateLimitOverTime != nil {
		values = append(values, *uk.RateLimitOverTime)
		fields = append(fields, fmt.Sprintf("rate_limit_over_time = $%d", counter))
		counter++
	}

	if uk.RateLimitUnit != nil {
		values = append(values, *uk.RateLimitUnit)
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

	if uk.PolicyIds != nil {
		values = append(values, sliceToSqlStringArray(*uk.PolicyIds))
		fields = append(fields, fmt.Sprintf("policy_ids = $%d", counter))
		counter++
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
		pq.Array(&k.PolicyIds),
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

func (s *Store) CreateKey(rk *key.RequestKey) (*key.ResponseKey, error) {
	query := `
		INSERT INTO keys (name, created_at, updated_at, tags, revoked, key_id, key, revoked_reason, cost_limit_in_usd, cost_limit_in_usd_over_time, cost_limit_in_usd_unit, rate_limit_over_time, rate_limit_unit, ttl, setting_id, allowed_paths, setting_ids, should_log_request, should_log_response, rotation_enabled, policy_ids)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
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
		rk.PolicyIds,
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
		pq.Array(&k.PolicyIds),
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
