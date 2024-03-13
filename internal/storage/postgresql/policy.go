package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
	"github.com/bricks-cloud/bricksllm/internal/policy"
	"github.com/lib/pq"
)

func (s *Store) CreatePolicyTable() error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS policies (
		id VARCHAR(255) PRIMARY KEY,
		created_at BIGINT NOT NULL,
		updated_at BIGINT NOT NULL,
		name VARCHAR(255) NOT NULL,
		tags VARCHAR(255)[],
		config JSONB NOT NULL,
		regex_config JSONB NOT NULL,
		custom_config JSONB NOT NULL
	)`

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	_, err := s.db.ExecContext(ctxTimeout, createTableQuery)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) CreatePolicy(p *policy.Policy) (*policy.Policy, error) {
	fields := []string{
		"id",
		"created_at",
		"updated_at",
		"tags",
		"name",
	}

	values := []any{
		p.Id,
		p.CreatedAt,
		p.UpdatedAt,
		pq.Array(p.Tags),
		p.Name,
	}

	vidxs := []string{
		"$1", "$2", "$3", "$4", "$5",
	}
	idx := 6

	if p.Config != nil {
		cd, err := json.Marshal(p.Config)
		if err != nil {
			return nil, err
		}

		fields = append(fields, "config")
		values = append(values, cd)
		vidxs = append(vidxs, fmt.Sprintf("$%d", idx))
		idx++
	}

	if p.RegexConfig != nil {
		cd, err := json.Marshal(p.RegexConfig)
		if err != nil {
			return nil, err
		}

		fields = append(fields, "regex_config")
		values = append(values, cd)
		vidxs = append(vidxs, fmt.Sprintf("$%d", idx))
		idx++
	}

	if p.CustomConfig != nil {
		cd, err := json.Marshal(p.CustomConfig)
		if err != nil {
			return nil, err
		}

		fields = append(fields, "custom_config")
		values = append(values, cd)
		vidxs = append(vidxs, fmt.Sprintf("$%d", idx))
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	query := fmt.Sprintf(`
	INSERT INTO policies (%s)
	VALUES (%s)
	RETURNING *
`, strings.Join(fields, ","), strings.Join(vidxs, ","))

	created := &policy.Policy{}

	var createdcd []byte
	var createdcusd []byte
	var createdregexd []byte
	row := s.db.QueryRowContext(ctx, query, values...)
	if err := row.Scan(
		&created.Id,
		&created.CreatedAt,
		&created.UpdatedAt,
		&created.Name,
		pq.Array(&created.Tags),
		&createdcd,
		&createdregexd,
		&createdcusd,
	); err != nil {

		return nil, err
	}

	if len(createdcd) != 0 {
		if err := json.Unmarshal(createdcd, &created.Config); err != nil {
			return nil, err
		}
	}

	if len(createdregexd) != 0 {
		if err := json.Unmarshal(createdregexd, &created.RegexConfig); err != nil {
			return nil, err
		}
	}

	if len(createdcusd) != 0 {
		if err := json.Unmarshal(createdcusd, &created.CustomConfig); err != nil {
			return nil, err
		}
	}

	return created, nil
}

func (s *Store) UpdatePolicy(id string, p *policy.UpdatePolicy) (*policy.Policy, error) {
	values := []any{
		id,
		p.UpdatedAt,
	}

	fields := []string{"updated_at = $2"}

	d := 3

	if len(p.Name) != 0 {
		values = append(values, p.Name)
		fields = append(fields, fmt.Sprintf("name = $%d", d))
		d++
	}

	if len(p.Tags) != 0 {
		values = append(values, pq.Array(p.Tags))
		fields = append(fields, fmt.Sprintf("tags = $%d", d))
		d++
	}

	if p.Config != nil {
		data, err := json.Marshal(p.Config)
		if err != nil {
			return nil, err
		}

		values = append(values, data)
		fields = append(fields, fmt.Sprintf("config = $%d", d))
		d++
	}

	if p.RegexConfig != nil {
		data, err := json.Marshal(p.RegexConfig)
		if err != nil {
			return nil, err
		}

		values = append(values, data)
		fields = append(fields, fmt.Sprintf("regex_config = $%d", d))
		d++
	}

	if p.CustomConfig != nil {
		data, err := json.Marshal(p.CustomConfig)
		if err != nil {
			return nil, err
		}

		values = append(values, data)
		fields = append(fields, fmt.Sprintf("custom_config = $%d", d))
	}

	query := fmt.Sprintf("UPDATE policies SET %s WHERE id = $1 RETURNING *", strings.Join(fields, ","))
	updated := &policy.Policy{}
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	var cd []byte
	var cusd []byte
	var regexd []byte
	row := s.db.QueryRowContext(ctxTimeout, query, values...)
	if err := row.Scan(
		&updated.Id,
		&updated.CreatedAt,
		&updated.UpdatedAt,
		&updated.Name,
		pq.Array(&updated.Tags),
		&cd,
		&regexd,
		&cusd,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, internal_errors.NewNotFoundError("policy is not found for id: " + id)
		}

		return nil, err
	}

	if len(cd) != 0 {
		if err := json.Unmarshal(cd, &updated.Config); err != nil {
			return nil, err
		}
	}

	if len(regexd) != 0 {
		if err := json.Unmarshal(regexd, &updated.RegexConfig); err != nil {
			return nil, err
		}
	}

	if len(cusd) != 0 {
		if err := json.Unmarshal(cusd, &updated.CustomConfig); err != nil {
			return nil, err
		}
	}

	return updated, nil
}

func (s *Store) GetAllPolicies() ([]*policy.Policy, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctxTimeout, "SELECT * FROM policies")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ps := []*policy.Policy{}
	for rows.Next() {
		var cd []byte
		var cusd []byte
		var regexd []byte

		p := &policy.Policy{}
		if err := rows.Scan(
			&p.Id,
			&p.CreatedAt,
			&p.UpdatedAt,
			&p.Name,
			pq.Array(&p.Tags),
			&cd,
			&regexd,
			&cusd,
		); err != nil {
			return nil, err
		}

		if len(cd) != 0 {
			if err := json.Unmarshal(cd, &p.Config); err != nil {
				return nil, err
			}
		}

		if len(regexd) != 0 {
			if err := json.Unmarshal(regexd, &p.RegexConfig); err != nil {
				return nil, err
			}
		}

		if len(cusd) != 0 {
			if err := json.Unmarshal(cusd, &p.CustomConfig); err != nil {
				return nil, err
			}
		}

		ps = append(ps, p)
	}

	return ps, nil
}

func (s *Store) GetPolicyById(id string) (*policy.Policy, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	row := s.db.QueryRowContext(ctxTimeout, "SELECT * FROM policies WHERE id = $1", id)
	p := &policy.Policy{}

	var cd []byte
	var cusd []byte
	var regexd []byte

	if err := row.Scan(
		&p.Id,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.Name,
		pq.Array(&p.Tags),
		&cd,
		&regexd,
		&cusd,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, internal_errors.NewNotFoundError("policy is not found for id: " + id)
		}

		return nil, err
	}

	if len(cd) != 0 {
		if err := json.Unmarshal(cd, &p.Config); err != nil {
			return nil, err
		}
	}

	if len(regexd) != 0 {
		if err := json.Unmarshal(regexd, &p.RegexConfig); err != nil {
			return nil, err
		}
	}

	if len(cusd) != 0 {
		if err := json.Unmarshal(cusd, &p.CustomConfig); err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (s *Store) GetPoliciesByTags(tags []string) ([]*policy.Policy, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	rows, err := s.db.QueryContext(ctxTimeout, "SELECT * FROM policies WHERE tags @> $1", pq.Array(tags))
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	ps := []*policy.Policy{}
	for rows.Next() {
		var cd []byte
		var cusd []byte
		var regexd []byte

		p := &policy.Policy{}

		if err := rows.Scan(
			&p.Id,
			&p.CreatedAt,
			&p.UpdatedAt,
			&p.Name,
			pq.Array(&p.Tags),
			&cd,
			&regexd,
			&cusd,
		); err != nil {
			return nil, err
		}

		if len(cd) != 0 {
			if err := json.Unmarshal(cd, &p.Config); err != nil {
				return nil, err
			}
		}

		if len(regexd) != 0 {
			if err := json.Unmarshal(regexd, &p.RegexConfig); err != nil {
				return nil, err
			}
		}

		if len(cusd) != 0 {
			if err := json.Unmarshal(cusd, &p.CustomConfig); err != nil {
				return nil, err
			}
		}

		ps = append(ps, p)

	}

	return ps, nil
}

func (s *Store) GetUpdatedPolicies(updatedAt int64) ([]*policy.Policy, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctxTimeout, "SELECT * FROM policies WHERE updated_at >= $1", updatedAt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ps := []*policy.Policy{}
	for rows.Next() {
		var cd []byte
		var cusd []byte
		var regexd []byte

		p := &policy.Policy{}
		if err := rows.Scan(
			&p.Id,
			&p.CreatedAt,
			&p.UpdatedAt,
			&p.Name,
			pq.Array(&p.Tags),
			&cd,
			&regexd,
			&cusd,
		); err != nil {
			return nil, err
		}

		if len(cd) != 0 {
			if err := json.Unmarshal(cd, &p.Config); err != nil {
				return nil, err
			}
		}

		if len(regexd) != 0 {
			if err := json.Unmarshal(regexd, &p.RegexConfig); err != nil {
				return nil, err
			}
		}

		if len(cusd) != 0 {
			if err := json.Unmarshal(cusd, &p.CustomConfig); err != nil {
				return nil, err
			}
		}

		ps = append(ps, p)
	}

	return ps, nil
}
