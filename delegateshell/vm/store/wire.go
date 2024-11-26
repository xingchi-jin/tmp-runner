package store

import (
	"github.com/drone-runners/drone-runner-aws/store"
	"github.com/drone-runners/drone-runner-aws/store/database"
	"github.com/drone-runners/drone-runner-aws/store/database/sql"
	"github.com/google/wire"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/jmoiron/sqlx"
)

var WireSet = wire.NewSet(
	ProvideInstanceStore,
	ProvideStageOwnerStore,
	ProvideSQLDatabase,
)

// ProvideSQLDatabase provides a database connection.
func ProvideSQLDatabase(config *delegate.Config) (*sqlx.DB, error) {
	// if a pool file is not set, don't connect to the database
	if config.VM.Pool.File == "" {
		return nil, nil
	}
	return database.ConnectSQL(
		config.VM.Database.Driver,
		config.VM.Database.Datasource,
	)
}

func ProvideInstanceStore(db *sqlx.DB) store.InstanceStore {
	if db == nil {
		return nil
	}
	return sql.NewInstanceStore(db)
}

func ProvideStageOwnerStore(db *sqlx.DB) store.StageOwnerStore {
	if db == nil {
		return nil
	}
	return sql.NewStageOwnerStore(db)
}
