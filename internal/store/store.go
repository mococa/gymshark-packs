package store

import "errors"

var (
	ErrLastPackSize     = errors.New("cannot remove the last pack size")
	ErrPackSizeNotFound = errors.New("pack size not found")
	ErrPackSizeExists   = errors.New("pack size already exists")
)

// PackSizeStore defines the interface for managing pack sizes.
type PackSizeStore interface {
	GetAll() ([]int, error)
	Add(size int) error
	Remove(size int) error
}
