package main

import (
	"context"

	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/facebookincubator/flog"
	"github.com/facebookincubator/nvdtools/vulndb/sqlutil"
)

const (
	ADD = iota
	UPDATE
	REMOVE
	GET
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
	
	if len(conf.CPEs) == 0 {
		return errors.New("Require at least one cpe to monitor")
	}

	ctx := context.Background()
	tx, err := db.BeginTx(ctx,nil)
	if err != nil {
		flog.Error(err)
		return errors.New("Internal error")
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
		flog.Error(err)
		return errors.New("Internal error")
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
		flog.Error(err)
		return err
	}

	return err
}


func UpdateConfiguration(db *sql.DB, conf Configuration) (err error) {
	
	if len(conf.CPEs) == 0 {
		flog.Error(err)
		return errors.New("Internal error")
	}

	ctx := context.Background()
	tx, err := db.BeginTx(ctx,nil)
	if err != nil {
		flog.Error(err)
		return errors.New("Internal error")
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	
	rows, err := tx.Query("SELECT COUNT(*) FROM monitored_configurations WHERE uid = ?", conf.Name)
	if err != nil {
		flog.Error(err)
		rows.Close()
		return errors.New("Internal error")
	}
	if rows.Next() != true {
		flog.Error(err)
		rows.Close()
		return errors.New("Internal error")
	}

	var count int
	err = rows.Scan(&count)
	if err != nil{
		flog.Error(err)
		rows.Close()
		return errors.New("Internal error")
	}
	if count != 1 {
		flog.Error(err)
		rows.Close()
		return errors.New("Configuration doesn't exist")
	}
	rows.Close()


	_, err = tx.Exec("DELETE FROM cpe_monitored WHERE configuration_uid = ?",conf.Name)
	if err != nil {
		flog.Error(err)
		return errors.New("Internal error")
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
		flog.Error(err)
		return errors.New("Internal error")
	}


	return err
}

func DeleteConfiguration(db *sql.DB, uid string) (err error) {
	
	ctx := context.Background()
	tx, err := db.BeginTx(ctx,nil)
	if err != nil {
		flog.Error(err)
		return errors.New("Internal error")
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else { err = tx.Commit()
		}
	}()
	
	rows, err := tx.Query("SELECT COUNT(*) FROM monitored_configurations WHERE uid = ?", uid)
	if err != nil {
		err = errors.Wrap(err, "Failed to check existence of configuration")
		rows.Close()
		return err
	}
	if rows.Next() != true {
		rows.Close()
		return errors.New("Internal error")
	}

	var count int
	err = rows.Scan(&count)
	if err != nil{
		if err != nil {
			flog.Error(err)
			rows.Close()
			return errors.New("Internal error")
		}
	}
	if count != 1 {
		rows.Close()
		return errors.New("Configuration doesn't exist")
	}
	rows.Close()


	_, err = tx.Exec("DELETE FROM cpe_monitored WHERE configuration_uid = ?",uid)
	if err != nil {
		flog.Error(err)
		return errors.New("Internal error")
	}

	_, err = tx.Exec("DELETE FROM monitored_configurations WHERE uid = ?",uid)
	if err != nil {
		flog.Error(err)
		return errors.New("Internal error")
	}
	
	return nil
}

func GetConfiguration(db *sql.DB, uid string) (Configuration, error) {
	config := Configuration {
		Name: uid,
		CPEs: nil,
	}

	q := sqlutil.Select("cpe_uri").
		From("cpe_monitored").
		Where(
			sqlutil.Cond().
				Equal("configuration_uid", uid),
		)
	
	query, args := q.String(), q.QueryArgs()

	rows, err := db.Query(query, args...)

	if err != nil {
		flog.Error(err)
		return config, errors.New("internal error")
	}
	defer rows.Close()

	for rows.Next() {
		var cpe string
		err = rows.Scan(&cpe)
		if err != nil {
			flog.Error(err)
			return config, errors.New("internal error")
		}
		config.CPEs = append(config.CPEs, cpe)
	}

	if len(config.CPEs) == 0 {
		return config, errors.New("Configuration doesn't exist")
	}
	
	return config, nil
}
