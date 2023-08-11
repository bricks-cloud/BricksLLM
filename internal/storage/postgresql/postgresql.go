package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/logger"
)

type Store struct {
	db *sql.DB
	lg logger.Logger
}

func NewStore(connStr string, lg logger.Logger) (*Store, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	return &Store{
		db: db,
		lg: lg,
	}, nil
}

func (s *Store) GetKeysByTag(tag string) ([]*key.ResponseKey, error) {
	rows, err := s.db.QueryContext(context.Background(), "SELECT * FROM keys WHERE $1 = ANY(tags)", tag)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := []*key.ResponseKey{}
	for rows.Next() {
		var k key.ResponseKey
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, &k)
	}

	return keys, nil
}

func (s *Store) UpdateKey(id string, rk *key.RequestKey) (*key.ResponseKey, error) {
	fields := []string{}
	counter := 2
	values := []any{
		id,
	}

	if len(rk.Name) != 0 {
		values = append(values, rk.Name)
		fields = append(fields, fmt.Sprintf("name = $%d", counter))
		counter++
	}

	if rk.UpdatedAt != 0 {
		values = append(values, rk.UpdatedAt)
		fields = append(fields, fmt.Sprintf("updated_at = $%d", counter))
		counter++
	}

	if len(rk.Tags) != 0 {
		values = append(values, rk.Tags)
		fields = append(fields, fmt.Sprintf("tags = $%d", counter))
		counter++
	}

	if rk.Revoked != nil {
		values = append(values, rk.Revoked)
		fields = append(fields, fmt.Sprintf("revoked = $%d", counter))
		counter++
	}

	if rk.Retrievable != nil {
		values = append(values, rk.Retrievable)
		fields = append(fields, fmt.Sprintf("retrievable = $%d", counter))
		counter++
	}

	if rk.CostLimitInUsd != 0 {
		values = append(values, rk.CostLimitInUsd)
		fields = append(fields, fmt.Sprintf("cost_limit_in_usd = $%d", counter))
		counter++
	}

	if rk.CostLimitInUsdPerDay != 0 {
		values = append(values, rk.CostLimitInUsdPerDay)
		fields = append(fields, fmt.Sprintf("cost_limit_in_usd_per_day = $%d", counter))
		counter++
	}

	if rk.RateLimit != 0 {
		values = append(values, rk.RateLimit)
		fields = append(fields, fmt.Sprintf("rate_limit = $%d", counter))
		counter++
	}

	if rk.RateLimitDuration != 0 {
		values = append(values, rk.RateLimit)
		fields = append(fields, fmt.Sprintf("rate_limit_duration = $%d", counter))
		counter++
	}

	if rk.Ttl != 0 {
		values = append(values, rk.Ttl)
		fields = append(fields, fmt.Sprintf("ttl = $%d", counter))
		counter++
	}

	query := fmt.Sprintf("UPDATE keys SET %s WHERE key_id = $1 RETURNING *;", strings.Join(fields, ","))

	var k key.ResponseKey
	if err := s.db.QueryRowContext(context.Background(), query, values...).Scan(&k); err != nil {
		return nil, err
	}

	return &k, nil
}

func (s *Store) CreateKey(rk *key.RequestKey) (*key.ResponseKey, error) {
	query := `
		INSERT INTO keys (name, created_at, updated_at, tags, key_id, revoked, key, retrievable, cost_limit_in_usd, cost_limit_in_usd_per_day, rate_limit, rate_limit_duration, ttl)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING *;
	`

	values := []any{
		rk.Name,
		rk.CreatedAt,
		rk.UpdatedAt,
		rk.Tags,
		rk.KeyId,
		rk.Revoked,
		rk.Key,
		rk.Retrievable,
		rk.CostLimitInUsd,
		rk.RateLimit,
		rk.RateLimitDuration.String(),
		rk.Ttl.String(),
	}

	var k key.ResponseKey
	if err := s.db.QueryRowContext(context.Background(), query, values...).Scan(&k); err != nil {
		return nil, err
	}

	return &k, nil
}

func (s *Store) DeleteKey(id string) error {
	_, err := s.db.ExecContext(context.Background(), "DELETE FROM keys WHERE key_id = $1", id)
	return err
}
