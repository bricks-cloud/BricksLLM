package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

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

func (s *Store) AlterProviderSettingsTable() error {
	alterTableQuery := `
		ALTER TABLE provider_settings ADD COLUMN IF NOT EXISTS name VARCHAR(255), ADD COLUMN IF NOT EXISTS allowed_models VARCHAR(255)[], ADD COLUMN IF NOT EXISTS cost_map JSONB NOT NULL DEFAULT '{}'::JSONB
	`

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	_, err := s.db.ExecContext(ctxTimeout, alterTableQuery)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) GetProviderSetting(id string, withSecret bool) (*provider.Setting, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	setting := &provider.Setting{}
	var data []byte
	var cmdata []byte
	var name sql.NullString
	err := s.db.QueryRowContext(ctxTimeout, "SELECT * FROM provider_settings WHERE $1 = id", id).Scan(
		&setting.Id,
		&setting.CreatedAt,
		&setting.UpdatedAt,
		&setting.Provider,
		&data,
		&name,
		pq.Array(&setting.AllowedModels),
		&cmdata,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, internal_errors.NewNotFoundError("provider setting is not found")
		}

		return nil, err
	}

	m := map[string]string{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	cm := &provider.CostMap{}
	if err := json.Unmarshal(cmdata, &cm); err != nil {
		return nil, err
	}

	if !withSecret {
		delete(m, "apikey")
	}

	setting.Setting = m
	setting.CostMap = cm

	setting.Name = name.String

	return setting, nil
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
		var cmdata []byte
		var name sql.NullString
		if err := rows.Scan(
			&setting.Id,
			&setting.CreatedAt,
			&setting.UpdatedAt,
			&setting.Provider,
			&data,
			&name,
			pq.Array(&setting.AllowedModels),
			&cmdata,
		); err != nil {
			return nil, err
		}

		m := map[string]string{}
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, err
		}

		cm := &provider.CostMap{}
		if err := json.Unmarshal(cmdata, &cm); err != nil {
			return nil, err
		}

		setting.Setting = m
		setting.CostMap = cm
		setting.Name = name.String
		settings = append(settings, setting)
	}

	return settings, nil
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

	if setting.CostMap != nil {
		data, err := json.Marshal(setting.CostMap)
		if err != nil {
			return nil, err
		}

		values = append(values, data)
		fields = append(fields, fmt.Sprintf("cost_map = $%d", d))
	}

	query := fmt.Sprintf("UPDATE provider_settings SET %s WHERE id = $1 RETURNING id, created_at, updated_at, provider, name, allowed_models, setting, cost_map;", strings.Join(fields, ","))
	updated := &provider.Setting{}
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	var rawd []byte
	var cmdata []byte

	row := s.db.QueryRowContext(ctxTimeout, query, values...)
	if err := row.Scan(
		&updated.Id,
		&updated.CreatedAt,
		&updated.UpdatedAt,
		&updated.Provider,
		&updated.Name,
		pq.Array(&updated.AllowedModels),
		&rawd,
		&cmdata,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, internal_errors.NewNotFoundError("provider setting is not found for: " + id)
		}

		return nil, err
	}

	m := map[string]string{}
	if err := json.Unmarshal(rawd, &m); err != nil {
		return nil, err
	}

	cm := &provider.CostMap{}
	if err := json.Unmarshal(cmdata, &cm); err != nil {
		return nil, err
	}

	delete(m, "apikey")

	updated.Setting = m
	updated.CostMap = cm

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
		INSERT INTO provider_settings (id, created_at, updated_at, provider, setting, name, allowed_models, cost_map)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at, provider, name, allowed_models, setting, cost_map
	`

	data, err := json.Marshal(setting.Setting)
	if err != nil {
		return nil, err
	}

	cmd, err := json.Marshal(setting.CostMap)
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
		cmd,
	}

	var rawd []byte
	var rawcmd []byte

	created := &provider.Setting{}
	var name sql.NullString
	if err := s.db.QueryRowContext(ctxTimeout, query, values...).Scan(
		&created.Id,
		&created.CreatedAt,
		&created.UpdatedAt,
		&created.Provider,
		&name,
		pq.Array(&created.AllowedModels),
		&rawd,
		&rawcmd,
	); err != nil {
		return nil, err
	}

	m := map[string]string{}
	if err := json.Unmarshal(rawd, &m); err != nil {
		return nil, err
	}

	cm := &provider.CostMap{}
	if err := json.Unmarshal(rawcmd, &cm); err != nil {
		return nil, err
	}

	delete(m, "apikey")

	created.Setting = m
	created.CostMap = cm

	created.Name = name.String
	return created, nil
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
		var cmdata []byte

		var name sql.NullString
		if err := rows.Scan(
			&setting.Id,
			&setting.CreatedAt,
			&setting.UpdatedAt,
			&setting.Provider,
			&data,
			&name,
			pq.Array(&setting.AllowedModels),
			&cmdata,
		); err != nil {
			return nil, err
		}

		m := map[string]string{}
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, err
		}

		cm := &provider.CostMap{}
		if err := json.Unmarshal(cmdata, &cm); err != nil {
			return nil, err
		}

		if !withSecret {
			delete(m, "apikey")
		}

		setting.Setting = m
		setting.CostMap = cm

		setting.Name = name.String
		settings = append(settings, setting)
	}

	if len(ids) != 0 && len(ids) != len(settings) {
		return nil, errors.New("not all settings are found")
	}

	return settings, nil
}
