package main

import (
	"context"

	"database/sql"
	"github.com/pkg/errors"

	"github.com/facebookincubator/flog"
	"github.com/facebookincubator/nvdtools/vulndb/sqlutil"
	_ "github.com/go-sql-driver/mysql"
)

type SearchResult struct {
	URI              string    `json:"uri" sql:"URI"`
	Vendor           string    `json:"vendor" sql:"vendor"`
	Product          string    `json:"product" sql:"product"`
	Title            string    `json:"title" sql:"title"`
	MinimunVersion   string    `json:"minimumVersion" sql:"min_version"`
}

func CPESearch(query string, start int,count int,db *sql.DB,ctx context.Context) ([]SearchResult, error) {
	if query == ""{
		return nil,nil
	}
	

	
	rows, err := db.Query(`
	SELECT 
		vendor,
		product,
		title,
		URI,
		MIN(version) AS min_version
	FROM cpe_dict
	WHERE MATCH(title) AGAINST(? IN NATURAL LANGUAGE MODE) 
	GROUP BY vendor, product
	LIMIT ? OFFSET ?
	`,query,count ,start)
	if err != nil {
		flog.Error(err)
		return nil, errors.New("internal error")
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var row SearchResult
		err = rows.Scan(sqlutil.NewRecordType(&row).Values()...)
		if err != nil {
			flog.Error(err)
			return nil, errors.New("internal error")
		}
		results = append(results, row)
	}


	return results,nil
}
