package store

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

// SQLiteStore implements PackSizeStore using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) the SQLite database at dbPath and seeds it
// with defaultSizes if the table is empty.
func NewSQLiteStore(dbPath string, defaultSizes []int) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if _, err = db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, err
	}

	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS pack_sizes (size INTEGER PRIMARY KEY)`); err != nil {
		return nil, err
	}

	s := &SQLiteStore{db: db}

	sizes, err := s.GetAll()
	if err != nil {
		return nil, err
	}
	if len(sizes) == 0 {
		for _, size := range defaultSizes {
			if err := s.Add(size); err != nil {
				return nil, err
			}
		}
	}

	return s, nil
}

func (s *SQLiteStore) GetAll() ([]int, error) {
	rows, err := s.db.Query("SELECT size FROM pack_sizes ORDER BY size ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sizes []int
	for rows.Next() {
		var size int
		if err := rows.Scan(&size); err != nil {
			return nil, err
		}
		sizes = append(sizes, size)
	}
	return sizes, rows.Err()
}

func (s *SQLiteStore) Add(size int) error {
	result, err := s.db.Exec("INSERT OR IGNORE INTO pack_sizes (size) VALUES (?)", size)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrPackSizeExists
	}
	return nil
}

// Remove deletes a pack size inside a transaction so the count-check and
// delete are atomic.
func (s *SQLiteStore) Remove(size int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var exists int
	if err := tx.QueryRow("SELECT COUNT(*) FROM pack_sizes WHERE size = ?", size).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return ErrPackSizeNotFound
	}

	var count int
	if err := tx.QueryRow("SELECT COUNT(*) FROM pack_sizes").Scan(&count); err != nil {
		return err
	}
	if count <= 1 {
		return ErrLastPackSize
	}

	if _, err := tx.Exec("DELETE FROM pack_sizes WHERE size = ?", size); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
