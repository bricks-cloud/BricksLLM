package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/user"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
	"github.com/lib/pq"
)

func (s *Store) CreateUsersTable() error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(255) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		created_at BIGINT NOT NULL,
		updated_at BIGINT NOT NULL,
		tags VARCHAR(255)[],
		revoked BOOLEAN NOT NULL,
		revoked_reason VARCHAR(255),
		cost_limit_in_usd FLOAT8,
		cost_limit_in_usd_over_time FLOAT8,
		cost_limit_in_usd_unit VARCHAR(255),
		rate_limit_over_time INT,
		rate_limit_unit VARCHAR(255),
		ttl VARCHAR(255),
		key_ids VARCHAR(255)[] NOT NULL DEFAULT ARRAY[]::VARCHAR(255)[],
		allowed_paths JSONB,
		allowed_models VARCHAR(255)[]
	)`

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	_, err := s.db.ExecContext(ctxTimeout, createTableQuery)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) CreateCreatedAtIndexForUsers() error {
	createIndexQuery := `
	CREATE INDEX IF NOT EXISTS created_at_idx ON users(created_at);
	`

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	_, err := s.db.ExecContext(ctxTimeout, createIndexQuery)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) GetUsers(tags, keyIds, userIds []string, offset, limit int) ([]*user.User, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	args := []any{}

	query := "SELECT * FROM users"

	if len(tags) != 0 || len(userIds) != 0 || len(keyIds) != 0 {
		query += " WHERE "
	}

	index := 1
	if len(tags) != 0 {
		args = append(args, pq.Array(tags))
		query += "tags @> $1"
		index += 1
	}

	if len(keyIds) != 0 {
		if index > 1 {
			query += " AND "
		}

		args = append(args, pq.Array(keyIds))
		query += fmt.Sprintf("key_ids @> ANY($%d)", index)
		index += 1
	}

	if len(userIds) != 0 {
		if index > 1 {
			query += " AND "
		}

		args = append(args, pq.Array(userIds))
		query += fmt.Sprintf("id = ANY($%d)", index)
		index += 1
	}

	if limit != 0 {
		query += fmt.Sprintf(" ORDER BY created_at DESC OFFSET %d LIMIT %d", offset, limit)
	}

	rows, err := s.db.QueryContext(ctxTimeout, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, internal_errors.NewNotFoundError("users are not found")
		}

		return nil, err
	}
	defer rows.Close()

	users := []*user.User{}
	for rows.Next() {
		var u user.User
		var data []byte
		if err := rows.Scan(
			&u.Id,
			&u.Name,
			&u.CreatedAt,
			&u.UpdatedAt,
			pq.Array(&u.Tags),
			&u.Revoked,
			&u.RevokedReason,
			&u.CostLimitInUsd,
			&u.CostLimitInUsdOverTime,
			&u.CostLimitInUsdUnit,
			&u.RateLimitOverTime,
			&u.RateLimitUnit,
			&u.Ttl,
			pq.Array(&u.KeyIds),
			&data,
			pq.Array(&u.AllowedModels),
		); err != nil {
			return nil, err
		}

		pu := &u

		if len(data) != 0 {
			pathConfigs := []key.PathConfig{}
			if err := json.Unmarshal(data, &pathConfigs); err != nil {
				return nil, err
			}

			pu.AllowedPaths = pathConfigs
		}

		users = append(users, pu)
	}

	return users, nil
}

func (s *Store) CreateUser(u *user.User) (*user.User, error) {
	query := `
		INSERT INTO users (id, name, created_at, updated_at, tags, revoked, revoked_reason, cost_limit_in_usd, cost_limit_in_usd_over_time, cost_limit_in_usd_unit, rate_limit_over_time, rate_limit_unit, ttl, key_ids, allowed_paths, allowed_models)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING *;
	`

	rdata, err := json.Marshal(u.AllowedPaths)
	if err != nil {
		return nil, err
	}

	values := []any{
		u.Id,
		u.Name,
		u.CreatedAt,
		u.UpdatedAt,
		pq.Array(u.Tags),
		false,
		"",
		u.CostLimitInUsd,
		u.CostLimitInUsdOverTime,
		u.CostLimitInUsdUnit,
		u.RateLimitOverTime,
		u.RateLimitUnit,
		u.Ttl,
		pq.Array(u.KeyIds),
		rdata,
		pq.Array(u.AllowedModels),
	}

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	var created user.User

	var data []byte
	if err := s.db.QueryRowContext(ctxTimeout, query, values...).Scan(
		&created.Id,
		&created.Name,
		&created.CreatedAt,
		&created.UpdatedAt,
		pq.Array(&created.Tags),
		&created.Revoked,
		&created.RevokedReason,
		&created.CostLimitInUsd,
		&created.CostLimitInUsdOverTime,
		&created.CostLimitInUsdUnit,
		&created.RateLimitOverTime,
		&created.RateLimitUnit,
		&created.Ttl,
		pq.Array(&created.KeyIds),
		&data,
		pq.Array(&created.AllowedModels),
	); err != nil {
		return nil, err
	}

	pu := &created

	if len(data) != 0 {
		pathConfigs := []key.PathConfig{}
		if err := json.Unmarshal(data, &pathConfigs); err != nil {
			return nil, err
		}

		pu.AllowedPaths = pathConfigs
	}

	return pu, nil
}

func (s *Store) UpdateUser(id string, uu *user.UpdateUser) (*user.User, error) {
	fields := []string{}
	counter := 2
	values := []any{
		id,
	}

	if len(uu.Name) != 0 {
		values = append(values, uu.Name)
		fields = append(fields, fmt.Sprintf("name = $%d", counter))
		counter++
	}

	if uu.UpdatedAt != 0 {
		values = append(values, uu.UpdatedAt)
		fields = append(fields, fmt.Sprintf("updated_at = $%d", counter))
		counter++
	}

	if len(uu.Tags) != 0 {
		values = append(values, pq.Array(uu.Tags))
		fields = append(fields, fmt.Sprintf("tags = $%d", counter))
		counter++
	}

	if uu.Revoked != nil {
		if *uu.Revoked && len(uu.RevokedReason) != 0 {
			values = append(values, uu.RevokedReason)
			fields = append(fields, fmt.Sprintf("revoked_reason = $%d", counter))
			counter++
		}

		if !*uu.Revoked {
			values = append(values, "")
			fields = append(fields, fmt.Sprintf("revoked_reason = $%d", counter))
			counter++
		}

		values = append(values, uu.Revoked)
		fields = append(fields, fmt.Sprintf("revoked = $%d", counter))
		counter++
	}

	if uu.CostLimitInUsd != nil {
		values = append(values, *uu.CostLimitInUsd)
		fields = append(fields, fmt.Sprintf("cost_limit_in_usd = $%d", counter))
		counter++
	}

	if uu.CostLimitInUsdOverTime != nil {
		values = append(values, *uu.CostLimitInUsdOverTime)
		fields = append(fields, fmt.Sprintf("cost_limit_in_usd_over_time = $%d", counter))
		counter++
	}

	if uu.CostLimitInUsdUnit != nil {
		values = append(values, *uu.CostLimitInUsdUnit)
		fields = append(fields, fmt.Sprintf("cost_limit_in_usd_unit = $%d", counter))
		counter++
	}

	if uu.RateLimitOverTime != nil {
		values = append(values, *uu.RateLimitOverTime)
		fields = append(fields, fmt.Sprintf("rate_limit_over_time = $%d", counter))
		counter++
	}

	if uu.RateLimitUnit != nil {
		values = append(values, *uu.RateLimitUnit)
		fields = append(fields, fmt.Sprintf("rate_limit_unit = $%d", counter))
		counter++
	}

	if uu.AllowedPaths != nil {
		data, err := json.Marshal(uu.AllowedPaths)
		if err != nil {
			return nil, err
		}

		values = append(values, data)
		fields = append(fields, fmt.Sprintf("allowed_paths = $%d", counter))
		counter++
	}

	if uu.KeyIds != nil {
		values = append(values, pq.Array(uu.KeyIds))
		fields = append(fields, fmt.Sprintf("key_ids = $%d", counter))
		counter++
	}

	if uu.AllowedModels != nil {
		values = append(values, pq.Array(uu.AllowedModels))
		fields = append(fields, fmt.Sprintf("allowed_models = $%d", counter))
		counter++
	}

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $1 RETURNING *;", strings.Join(fields, ","))

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	var updated user.User

	var data []byte
	if err := s.db.QueryRowContext(ctxTimeout, query, values...).Scan(
		&updated.Id,
		&updated.Name,
		&updated.CreatedAt,
		&updated.UpdatedAt,
		pq.Array(&updated.Tags),
		&updated.Revoked,
		&updated.RevokedReason,
		&updated.CostLimitInUsd,
		&updated.CostLimitInUsdOverTime,
		&updated.CostLimitInUsdUnit,
		&updated.RateLimitOverTime,
		&updated.RateLimitUnit,
		&updated.Ttl,
		pq.Array(&updated.KeyIds),
		&data,
		pq.Array(&updated.AllowedModels),
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, internal_errors.NewNotFoundError(fmt.Sprintf("key not found for id: %s", id))
		}
		return nil, err
	}

	pu := &updated

	if len(data) != 0 {
		pathConfigs := []key.PathConfig{}
		if err := json.Unmarshal(data, &pathConfigs); err != nil {
			return nil, err
		}

		updated.AllowedPaths = pathConfigs
	}

	return pu, nil
}
