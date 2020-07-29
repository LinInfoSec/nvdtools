package main

import (
	"context"

	"database/sql"
	"github.com/pkg/errors"

	"github.com/facebookincubator/flog"
	"github.com/facebookincubator/nvdtools/vulndb/sqlutil"
	_ "github.com/go-sql-driver/mysql"
)

type InfoResult struct {
	Total int       `json:"total"`
	Data  []CPEInfo `json:"data"`
}

type CPEInfo struct {
	URI        string      `json:"uri"`
	Part       string      `json:"part"`
	Vendor     string      `json:"vendor"`
	Product    string      `json:"product"`
	Version    string      `json:"version"`
	Updatecl   string      `json:"updatecl"`
	Edition    string      `json:"edition"`
	Swedition  string      `json:"swedition"`
	TargetSW   string      `json:"targetSW"`
	Targethw   string      `json:"targethw"`
	Other      string      `json:"other"`
	Language   string      `json:"language"`
	Title      string      `json:"title"`
	References []Reference `json:"references"`
}

type Reference struct {
	URL         string `json:"url" sql:"url"`
	Description string `json:"description" sql:"description"`
}

type InfoData struct {
	vendor  string
	product string
	part    string
	start   int
	count   int
}

func getReferences(uri string, db *sql.DB, ctx context.Context) ([]Reference, error) {
	references := []Reference{}

	r := sqlutil.NewRecordType(Reference{})
	q := sqlutil.Select(r.Fields()...).
		From("cpe_references").
		Where(
			sqlutil.Cond().
				Equal("cpe_uri", uri),
		)

	query, args := q.String(), q.QueryArgs()

	rows, err := db.Query(query, args...)
	if err != nil {
		flog.Error(err)
		return nil, errors.New("internal error")
	}
	defer rows.Close()

	for rows.Next() {
		var sr Reference
		err = rows.Scan(sqlutil.NewRecordType(&sr).Values()...)
		if err != nil {
			return nil, errors.Wrap(err, "cannot scan snooze data")
		}
		references = append(references, sr)
	}

	return references, nil
}

func getCpes(data InfoData, db *sql.DB, ctx context.Context) (InfoResult, error) {

	infoResults := InfoResult{
		Total: 0,
		Data:  nil,
	}

	q := sqlutil.Select("COUNT(*)").
		From("cpe_dict").
		Where(
			sqlutil.Cond().
				Equal("part", data.part).
				And().
				Equal("product", data.product).
				And().
				Equal("vendor", data.vendor),
		)

	query, args := q.String(), q.QueryArgs()

	rows, err := db.Query(query, args...)

	if err != nil || !rows.Next() {
		flog.Error(err)
		return infoResults, errors.New("internal error")
	}

	err = rows.Scan(&infoResults.Total)
	rows.Close()
	if err != nil {
		flog.Error(err)
		return infoResults, errors.New("internal error")
	}

	rows, err = db.Query(`
		SELECT
			URI,
			part,
			vendor,
			product,
			version,
			updatecl,
			edition,
			swedition,
			targetSW,
			targethw,
			other,
			language,
			title
		FROM 
			cpe_dict
		WHERE
			part = ? AND
			product = ? AND
			vendor = ?
		ORDER BY version
		LIMIT ? OFFSET ?
	`, data.part, data.product, data.vendor, data.count, data.start)
	if err != nil {
		flog.Error(err)
		return infoResults, errors.New("internal error")
	}
	defer rows.Close()

	for rows.Next() {
		var sr CPEInfo
		err = rows.Scan(
			&sr.URI,
			&sr.Part,
			&sr.Vendor,
			&sr.Product,
			&sr.Version,
			&sr.Updatecl,
			&sr.Edition,
			&sr.Swedition,
			&sr.TargetSW,
			&sr.Targethw,
			&sr.Other,
			&sr.Language,
			&sr.Title,
		)

		if err != nil {
			flog.Error(err)
			return infoResults, errors.New("internal error")
		}
		sr.References, err = getReferences(sr.URI, db, ctx)
		if err != nil {
			flog.Error(err)
			return infoResults, errors.New("internal error")
		}

		infoResults.Data = append(infoResults.Data, sr)
	}
	return infoResults, nil

}
