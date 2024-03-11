package postgresql

import "context"

func (s *Store) CreatePolicyTable() error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS provider_settings (
		id VARCHAR(255) PRIMARY KEY,
		created_at BIGINT NOT NULL,
		updated_at BIGINT NOT NULL,
		config JSONB NOT NULL
		regex_config JSONB NOT NULL
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
