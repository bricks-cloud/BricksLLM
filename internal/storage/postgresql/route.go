package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
	"github.com/bricks-cloud/bricksllm/internal/route"
	"github.com/lib/pq"
)

func (s *Store) CreateRoutesTable() error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS routes (
		id VARCHAR(255) PRIMARY KEY,
		created_at BIGINT NOT NULL,
		updated_at BIGINT NOT NULL,
		name VARCHAR(255) NOT NULL,
		path VARCHAR(255) NOT NULL,
		key_ids VARCHAR(255)[] NOT NULL,
		steps JSONB NOT NULL,
		cache_config JSONB NOT NULL
	)`

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	_, err := s.db.ExecContext(ctxTimeout, createTableQuery)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) AlterRoutesTable() error {
	alterTableQuery := `
		ALTER TABLE routes ADD COLUMN IF NOT EXISTS request_format VARCHAR(255) NOT NULL DEFAULT '', ADD COLUMN IF NOT EXISTS retry_strategy VARCHAR(255) NOT NULL DEFAULT '';
	`

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	_, err := s.db.ExecContext(ctxTimeout, alterTableQuery)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) DeleteRoute(id string) error {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	if _, err := s.db.ExecContext(ctxTimeout, "DELETE FROM routes WHERE $1 = id", id); err != nil {
		if err == sql.ErrNoRows {
			return internal_errors.NewNotFoundError("no rows")
		}
		return err
	}

	return nil
}

func (s *Store) CreateRoute(r *route.Route) (*route.Route, error) {
	sbytes, err := json.Marshal(r.Steps)
	if err != nil {
		return nil, err
	}

	cbytes, err := json.Marshal(r.CacheConfig)
	if err != nil {
		return nil, err
	}

	values := []any{
		r.Id,
		r.CreatedAt,
		r.UpdatedAt,
		r.Name,
		r.Path,
		sliceToSqlStringArray(r.KeyIds),
		sbytes,
		cbytes,
		r.RequestFormat,
		r.RetryStrategy,
	}

	query := `
	INSERT INTO routes (id, created_at, updated_at, name, path, key_ids, steps, cache_config, request_format, retry_strategy)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	RETURNING id, created_at, updated_at, name, path, key_ids, steps, cache_config, request_format, retry_strategy
`

	created := &route.Route{}

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	var cdata []byte
	var sdata []byte

	if err := s.db.QueryRowContext(ctxTimeout, query, values...).Scan(
		&created.Id,
		&created.CreatedAt,
		&created.UpdatedAt,
		&created.Name,
		&created.Path,
		pq.Array(&created.KeyIds),
		&sdata,
		&cdata,
		&created.RequestFormat,
		&created.RetryStrategy,
	); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(sdata, &created.Steps); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(cdata, &created.CacheConfig); err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Store) GetRoute(id string) (*route.Route, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	var cdata []byte
	var sdata []byte

	created := &route.Route{}
	if err := s.db.QueryRowContext(ctxTimeout, "SELECT * FROM routes WHERE $1 = id", id).Scan(
		&created.Id,
		&created.CreatedAt,
		&created.UpdatedAt,
		&created.Name,
		&created.Path,
		pq.Array(&created.KeyIds),
		&sdata,
		&cdata,
		&created.RequestFormat,
		&created.RetryStrategy,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, internal_errors.NewNotFoundError("custom provider is not found")
		}

		return nil, err
	}

	if err := json.Unmarshal(sdata, &created.Steps); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(cdata, &created.CacheConfig); err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Store) GetRouteByPath(path string) (*route.Route, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	var cdata []byte
	var sdata []byte

	created := &route.Route{}
	if err := s.db.QueryRowContext(ctxTimeout, "SELECT * FROM routes WHERE $1 = path", path).Scan(
		&created.Id,
		&created.CreatedAt,
		&created.UpdatedAt,
		&created.Name,
		&created.Path,
		pq.Array(&created.KeyIds),
		&sdata,
		&cdata,
		&created.RequestFormat,
		&created.RetryStrategy,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, internal_errors.NewNotFoundError("route is not found")
		}

		return nil, err
	}

	if err := json.Unmarshal(sdata, &created.Steps); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(cdata, &created.CacheConfig); err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Store) GetUpdatedRoutes(updatedAt int64) ([]*route.Route, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctxTimeout, "SELECT * FROM routes WHERE updated_at >= $1", updatedAt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	routes := []*route.Route{}
	for rows.Next() {
		r := &route.Route{}
		var cdata []byte
		var sdata []byte

		if err := rows.Scan(
			&r.Id,
			&r.CreatedAt,
			&r.UpdatedAt,
			&r.Name,
			&r.Path,
			pq.Array(&r.KeyIds),
			&sdata,
			&cdata,
			&r.RequestFormat,
			&r.RetryStrategy,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(sdata, &r.Steps); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(cdata, &r.CacheConfig); err != nil {
			return nil, err
		}

		routes = append(routes, r)
	}

	return routes, nil
}

func (s *Store) GetRoutes() ([]*route.Route, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctxTimeout, "SELECT * FROM routes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	routes := []*route.Route{}
	for rows.Next() {
		r := &route.Route{}
		var cdata []byte
		var sdata []byte

		if err := rows.Scan(
			&r.Id,
			&r.CreatedAt,
			&r.UpdatedAt,
			&r.Name,
			&r.Path,
			pq.Array(&r.KeyIds),
			&sdata,
			&cdata,
			&r.RequestFormat,
			&r.RetryStrategy,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(sdata, &r.Steps); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(cdata, &r.CacheConfig); err != nil {
			return nil, err
		}

		routes = append(routes, r)
	}

	return routes, nil
}
