package main

import (
	"context"

	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/facebookincubator/nvdtools/vulndb/sqlutil"
)

const (
	ADD = iota
	UPDATE
	REMOVE
)

type Configuration struct {
	Name string `json:"configuration"`
	CPEs []string `json:"cpes"`
}

type ConfigurationRecord struct {
	URI                string `sql:"cpe_uri"`
	Configuration      string `sql:"configuration_uid"`
}

func AddConfiguration(db *sql.DB, conf Configuration) (err error) {
	
	if len(conf.cpes) == 0 {
		return errors.New("Require at least one cpe to monitor")
	}

	ctx := context.Background()
	tx, err := db.BeginTx(ctx,nil)
	if err != nil {
		err = errors.Wrap(err, "Failed to begin configuration transaction")
		return
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	
	_, err = tx.Exec("INSERT INTO monitored_configurations (uid) VALUES (?)", conf.Name)
	if err != nil {
		err = errors.Wrap(err, "Failed to insert values")
		return
	}

	toInsert := []ConfigurationRecord{}

	for _, cpe := range conf.CPEs {
		rec := ConfigurationRecord {
			URI: cpe,
			Configuration: conf.Name,
		}
		toInsert = append(toInsert, rec)
	}

	records := sqlutil.NewRecords(toInsert)
	q := sqlutil.Insert().
		Into("cpe_monitored").
		Fields(records.Fields()...).
		Values(records...)

	query, args  := q.String(), q.QueryArgs()
	_, err = tx.Exec(query, args...)
	if err != nil {
		err = errors.Wrap(err, "Cannot insert configuration")
		return
	}

	return
}


