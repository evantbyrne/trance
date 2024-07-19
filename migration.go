package trance

import (
	"errors"
	"fmt"
	"reflect"
	"time"
)

type Migration interface {
	Down() error
	Up() error
}

type MigrationLogs struct {
	CreatedAt     time.Time `@:"created_at"`
	Direction     string    `@:"direction" @length:"10"`
	Id            int64     `@:"id" @primary:"true"`
	MigrationType string    `@:"migration_type" @length:"255"`
}

func MigrateDown(migrations []Migration) ([]string, error) {
	defer PurgeWeaves()
	logs, latestIndex, err := migrateSetup(migrations)
	if err != nil {
		return logs, err
	}

	for i := latestIndex; i > -1; i-- {
		PurgeWeaves()
		migrationType := reflect.TypeOf(migrations[i]).String()
		logs = append(logs, "Migrating down to "+migrationType+"...")
		if err := migrations[i].Down(); err != nil {
			return logs, errors.Join(fmt.Errorf("migration %s: failed", migrationType), err)
		}
		err := Query[MigrationLogs]().Insert(&MigrationLogs{
			CreatedAt:     time.Now(),
			Direction:     "down",
			MigrationType: migrationType,
		}).Error
		if err != nil {
			return logs, errors.Join(fmt.Errorf("migration %s: failed to insert migration logs", migrationType), err)
		}
	}

	return logs, nil
}

func migrateSetup(migrations []Migration) ([]string, int, error) {
	logs := make([]string, 0)

	err := Query[MigrationLogs]().TableCreate(TableCreateConfig{IfNotExists: true}).Error
	if err != nil {
		return nil, -1, errors.Join(errors.New("trance: migrations setup: failed to create table for migration logs"), err)
	}

	latest, err := Query[MigrationLogs]().Sort("-id").CollectFirst()
	latestIndex := -1
	if err != nil {
		if _, notFound := err.(ErrorNotFound); !notFound {
			return nil, -1, errors.Join(errors.New("trance: migrations setup: failed to get migrations list:"), err)
		}
	} else {
		for i, migration := range migrations {
			if latest.MigrationType == reflect.TypeOf(migration).String() {
				if latest.Direction == "down" {
					latestIndex = i - 1
				} else {
					latestIndex = i
				}
				break
			}
		}
	}

	return logs, latestIndex, nil
}

func MigrateUp(migrations []Migration) ([]string, error) {
	defer PurgeWeaves()
	logs, latestIndex, err := migrateSetup(migrations)
	if err != nil {
		return logs, err
	}

	for i := latestIndex + 1; i < len(migrations); i++ {
		PurgeWeaves()
		migrationType := reflect.TypeOf(migrations[i]).String()
		logs = append(logs, "Migrating up to "+migrationType+"...")
		if err := migrations[i].Up(); err != nil {
			return logs, errors.Join(fmt.Errorf("trance: migration %s: failed", migrationType), err)
		}
		err := Query[MigrationLogs]().Insert(&MigrationLogs{
			CreatedAt:     time.Now(),
			Direction:     "up",
			MigrationType: migrationType,
		}).Error
		if err != nil {
			return logs, errors.Join(fmt.Errorf("trance: migration %s: failed to insert migration logs", migrationType), err)
		}
	}

	return logs, nil
}
