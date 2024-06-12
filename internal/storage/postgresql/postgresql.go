package postgresql

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"time"

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
