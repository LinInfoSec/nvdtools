package main

import (
	"context"

	"database/sql"
	"github.com/pkg/errors"

	"github.com/facebookincubator/flog"
	_ "github.com/go-sql-driver/mysql"
)

type SearchResult struct {
	Total int         `json:"total"`
	Data  []SearchRow `'json:"data"`
}

type SearchRow struct {
	URI            string `json:"uri" sql:"URI"`
	Part           string `jswon:"uri" sql:"part"`
	Vendor         string `json:"vendor" sql:"vendor"`
	Product        string `json:"product" sql:"product"`
	Title          string `json:"title" sql:"title"`
	MinimunVersion string `json:"minimumVersion" sql:"min_version"`
}

type searchData struct {
	query string
	start int
	count int
}

func CPESearch(data searchData, db *sql.DB, ctx context.Context) (SearchResult, error) {

	searchResults := SearchResult{
		Total: 0,
		Data:  nil,
	}

	if data.query == "" {
		return searchResults, nil
	}

	rows, err := db.Query(`
	SELECT 
		COUNT(DISTINCT part, vendor, product)
	FROM cpe_dict
	WHERE MATCH(title) AGAINST(? IN NATURAL LANGUAGE MODE) 
	`, data.query)

	if err != nil || !rows.Next() {
		flog.Error(err)
		return searchResults, errors.New("internal error")
	}

	err = rows.Scan(&searchResults.Total)
	rows.Close()
	if err != nil {
		flog.Error(err)
		return searchResults, errors.New("internal error")
	}

	rows, err = db.Query(`
	SELECT 
		URI,
		part,
		vendor,
		product,
		title,
		MIN(version) AS min_version
	FROM cpe_dict
	WHERE MATCH(title) AGAINST(? IN NATURAL LANGUAGE MODE) 
	GROUP BY vendor, product, part
	LIMIT ? OFFSET ?
	`, data.query, data.count, data.start)
	if err != nil {
		flog.Error(err)
		return searchResults, errors.New("internal error")
	}
	defer rows.Close()

	for rows.Next() {
		var row SearchRow
		err = rows.Scan(&row.URI, &row.Part, &row.Vendor, &row.Product, &row.Title, &row.MinimunVersion)
		if err != nil {
			flog.Error(err)
			return searchResults, errors.New("internal error")
		}
		searchResults.Data = append(searchResults.Data, row)
	}

	return searchResults, nil
}
