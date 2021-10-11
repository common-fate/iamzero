package storage

import (
	"github.com/asdine/storm/v3"
	"github.com/jmoiron/sqlx"
)

// StorageFactory builds and configures the storage
// layer of the application
type Storage struct {
	Event   EventStorage
	Finding FindingStorage
	Action  ActionStorage
}

// BuildPostgresStorage builds the storage layer with Postgres as the driver
func BuildPostgresStorage(db *sqlx.DB) *Storage {
	return &Storage{
		Event:   NewPostgresEventStorage(db),
		Finding: NewPostgresFindingStorage(db),
		Action:  NewPostgresActionStorage(db),
	}
}

// BuildBoltStorage builds the storage layer with BoltDB as the driver
func BuildBoltStorage(db *storm.DB) *Storage {
	return &Storage{
		Event:   &NoOpEventStorage{}, // currently unused in local workflows, so we pass the no-op.
		Finding: NewBoltFindingStorage(db),
		Action:  NewBoltActionStorage(db),
	}
}

// BuildBoltStorage builds the storage layer using in-memory arrays
func BuildInMemoryStorage() *Storage {
	return &Storage{
		Event:   &NoOpEventStorage{}, // currently unused in local workflows, so we pass the no-op.
		Finding: NewInMemoryFindingStorage(),
		Action:  NewInMemoryActionStorage(),
	}
}
