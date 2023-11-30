package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
	"github.com/bricks-cloud/bricksllm/internal/provider/custom"
)

func (s *Store) CreateCustomProvidersTable() error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS custom_providers (
		id VARCHAR(255) PRIMARY KEY,
		created_at BIGINT NOT NULL,
		updated_at BIGINT NOT NULL,
		provider VARCHAR(255) NOT NULL,
		route_configs JSONB NOT NULL,
		authentication_param VARCHAR(255) NOT NULL
	)`

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()
	_, err := s.db.ExecContext(ctxTimeout, createTableQuery)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) DropCustomProvidersTable() error {
	dropTableQuery := `DROP TABLE custom_providers`
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	_, err := s.db.ExecContext(ctxTimeout, dropTableQuery)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) CreateCustomProvider(provider *custom.Provider) (*custom.Provider, error) {
	query := `
		INSERT INTO custom_providers (id, created_at, updated_at, provider, route_configs, authentication_param)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at, provider, route_configs, authentication_param
	`

	bytes, err := json.Marshal(provider.RouteConfigs)
	if err != nil {
		return nil, err
	}

	values := []any{
		provider.Id,
		provider.CreatedAt,
		provider.UpdatedAt,
		provider.Provider,
		bytes,
		provider.AuthenticationParam,
	}

	created := &custom.Provider{}

	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	var data []byte
	if err := s.db.QueryRowContext(ctxTimeout, query, values...).Scan(
		&created.Id,
		&created.CreatedAt,
		&created.UpdatedAt,
		&created.Provider,
		&data,
		&created.AuthenticationParam,
	); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &created.RouteConfigs); err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Store) GetCustomProviderByName(name string) (*custom.Provider, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	retrieved := &custom.Provider{}
	var data []byte
	if err := s.db.QueryRowContext(ctxTimeout, "SELECT * FROM custom_providers WHERE $1 = provider", name).Scan(
		&retrieved.Id,
		&retrieved.CreatedAt,
		&retrieved.UpdatedAt,
		&retrieved.Provider,
		&data,
		&retrieved.AuthenticationParam,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, internal_errors.NewNotFoundError("custom provider is not found")
		}
		return nil, err
	}

	return retrieved, nil
}

func (s *Store) GetCustomProvider(id string) (*custom.Provider, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	retrieved := &custom.Provider{}
	var data []byte
	if err := s.db.QueryRowContext(ctxTimeout, "SELECT * FROM custom_providers WHERE $1 = id", id).Scan(
		&retrieved.Id,
		&retrieved.CreatedAt,
		&retrieved.UpdatedAt,
		&retrieved.Provider,
		&data,
		&retrieved.AuthenticationParam,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, internal_errors.NewNotFoundError("custom provider is not found")
		}
		return nil, err
	}

	if err := json.Unmarshal(data, &retrieved.RouteConfigs); err != nil {
		return nil, err
	}

	return retrieved, nil
}

func (s *Store) GetCustomProviders() ([]*custom.Provider, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctxTimeout, "SELECT * FROM custom_providers")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	providers := []*custom.Provider{}
	for rows.Next() {
		provider := &custom.Provider{}
		var data []byte
		if err := rows.Scan(
			&provider.Id,
			&provider.CreatedAt,
			&provider.UpdatedAt,
			&provider.Provider,
			&data,
			&provider.AuthenticationParam,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(data, &provider.RouteConfigs); err != nil {
			return nil, err
		}

		providers = append(providers, provider)
	}

	return providers, nil
}

func (s *Store) UpdateCustomProvider(id string, provider *custom.UpdateProvider) (*custom.Provider, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	retrieved, err := s.GetCustomProvider(id)
	if err != nil {
		return nil, err
	}

	fields := []string{}
	counter := 2
	values := []any{
		id,
	}

	if provider.AuthenticationParam != nil {
		values = append(values, provider.AuthenticationParam)
		fields = append(fields, fmt.Sprintf("authentication_param = $%d", counter))
		counter++
	}

	if provider.UpdatedAt != 0 {
		values = append(values, provider.UpdatedAt)
		fields = append(fields, fmt.Sprintf("updated_at = $%d", counter))
		counter++
	}

	if len(provider.RouteConfigs) != 0 {
		merged := mergeRouterConfigs(retrieved.RouteConfigs, provider.RouteConfigs)
		bytes, err := json.Marshal(merged)
		if err != nil {
			return nil, err
		}

		values = append(values, bytes)
		fields = append(fields, fmt.Sprintf("route_configs = $%d", counter))
		counter++
	}

	query := fmt.Sprintf("UPDATE custom_providers SET %s WHERE $1 = id RETURNING *", strings.Join(fields, ","))

	ctxTimeout, cancel = context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	updated := &custom.Provider{}
	var updatedData []byte

	if err := s.db.QueryRowContext(ctxTimeout, query, values...).Scan(
		&updated.Id,
		&updated.CreatedAt,
		&updated.UpdatedAt,
		&updated.Provider,
		&updatedData,
		&updated.AuthenticationParam,
	); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(updatedData, &updated.RouteConfigs); err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Store) GetUpdatedCustomProviders(updatedAt int64) ([]*custom.Provider, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	rows, err := s.db.QueryContext(ctxTimeout, "SELECT * FROM custom_providers WHERE updated_at >= $1", updatedAt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	providers := []*custom.Provider{}
	for rows.Next() {
		provider := &custom.Provider{}
		var data []byte

		if err := rows.Scan(
			&provider.Id,
			&provider.CreatedAt,
			&provider.UpdatedAt,
			&provider.Provider,
			&data,
			&provider.AuthenticationParam,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(data, &provider.RouteConfigs); err != nil {
			return nil, err
		}

		providers = append(providers, provider)
	}

	return providers, nil
}

func mergeRouterConfigs(existingConfigs []*custom.RouteConfig, targetConfigs []*custom.RouteConfig) []*custom.RouteConfig {
	result := []*custom.RouteConfig{}

	pathToRouteMap := map[string]*custom.RouteConfig{}
	for _, existing := range existingConfigs {
		pathToRouteMap[existing.Path] = existing
	}

	for _, target := range targetConfigs {
		existing, ok := pathToRouteMap[target.Path]
		if !ok {
			pathToRouteMap[target.Path] = target
			continue
		}

		merged := &custom.RouteConfig{
			Path: existing.Path,
		}

		if len(target.StreamLocation) != 0 {
			merged.StreamLocation = target.StreamLocation
		}

		if len(target.StreamLocation) == 0 {
			merged.StreamLocation = existing.StreamLocation
		}

		if len(target.ModelLocation) != 0 {
			merged.ModelLocation = target.ModelLocation
		}

		if len(target.ModelLocation) == 0 {
			merged.ModelLocation = existing.ModelLocation
		}

		if len(target.RequestPromptLocation) != 0 {
			merged.RequestPromptLocation = target.RequestPromptLocation
		}

		if len(target.RequestPromptLocation) == 0 {
			merged.RequestPromptLocation = existing.RequestPromptLocation
		}

		if len(target.ResponseCompletionLocation) != 0 {
			merged.ResponseCompletionLocation = target.ResponseCompletionLocation
		}

		if len(target.ResponseCompletionLocation) == 0 {
			merged.ResponseCompletionLocation = existing.ResponseCompletionLocation
		}

		if len(target.StreamEndWord) != 0 {
			merged.StreamEndWord = target.StreamEndWord
		}

		if len(target.StreamEndWord) == 0 {
			merged.StreamEndWord = existing.StreamEndWord
		}

		if len(target.StreamResponseCompletionLocation) != 0 {
			merged.StreamResponseCompletionLocation = target.StreamResponseCompletionLocation
		}

		if len(target.StreamResponseCompletionLocation) == 0 {
			merged.StreamResponseCompletionLocation = existing.StreamResponseCompletionLocation
		}

		if target.StreamMaxEmptyMessages != 0 {
			merged.StreamMaxEmptyMessages = target.StreamMaxEmptyMessages
		}

		if target.StreamMaxEmptyMessages == 0 {
			merged.StreamMaxEmptyMessages = existing.StreamMaxEmptyMessages
		}

		if len(target.StreamResponseCompletionLocation) == 0 {
			merged.StreamResponseCompletionLocation = existing.StreamResponseCompletionLocation
		}

		if len(target.TargetUrl) != 0 {
			merged.TargetUrl = target.TargetUrl
		}

		if len(target.TargetUrl) == 0 {
			merged.TargetUrl = existing.TargetUrl
		}

		pathToRouteMap[merged.Path] = merged
	}

	for _, v := range pathToRouteMap {
		result = append(result, v)
	}

	return result
}
