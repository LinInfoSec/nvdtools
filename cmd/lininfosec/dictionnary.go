package main

import (
	"compress/gzip"
	"context"
	"os"
	"time"

	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/facebookincubator/flog"
	"github.com/facebookincubator/nvdtools/cpedict"
	"github.com/facebookincubator/nvdtools/providers/nvd"
	"github.com/facebookincubator/nvdtools/vulndb/sqlutil"
	"github.com/facebookincubator/nvdtools/wfn"
)

// CPERecord represents a db record of the `cpe_dict` table.
type CPERecord struct {
	Part      string `sql:"Part"`
	Vendor    string `sql:"Vendor"`
	Product   string `sql:"Product"`
	Version   string `sql:"Version"`
	Update    string `sql:"UpdateCl"`
	Edition   string `sql:"Edition"`
	SWEdition string `sql:"SWEdition"`
	TargetSW  string `sql:"TargetSW"`
	TargetHW  string `sql:"TargetHW"`
	Other     string `sql:"Other"`
	Language  string `sql:"Language"`
	Title     string `sql:"title"`
	URI       string `sql:"URI"`
}

// ReferenceRecord represents a db record of the `cpe_references` table.
type ReferenceRecord struct {
	URI         string `sql:"cpe_uri"`
	URL         string `sql:"url"`
	Description string `sql:"description"`
}

// Insert a batch of CPEs into the database
func insertBatch(batch []CPERecord, references []ReferenceRecord, db *sql.DB, ctx context.Context) error {
	if len(batch) != 0 {
		records := sqlutil.NewRecords(batch)
		q := sqlutil.Replace().
			Into("cpe_dict").
			Fields(records.Fields()...).
			Values(records...)

		query, args := q.String(), q.QueryArgs()
		_, err := db.ExecContext(ctx, query, args...)
		if err != nil {
			return errors.Wrap(err, "Cannot insert CPE batch")
		}
	}

	if len(references) != 0 {
		records := sqlutil.NewRecords(references)
		q := sqlutil.Replace().
			Into("cpe_references").
			Fields(records.Fields()...).
			Values(records...)

		query, args := q.String(), q.QueryArgs()
		_, err := db.ExecContext(ctx, query, args...)
		if err != nil {
			return errors.Wrap(err, "Cannot insert CPE references")
		}
	}

	return nil
}

func ImportDictionnary(db *sql.DB, ctx context.Context) error {

	var feed nvd.CPE
	feed.Set("cpe-2.3.xml.gz")

	source := nvd.NewSourceConfig()
	dfs := nvd.Sync{
		Feeds:    []nvd.Syncer{feed},
		Source:   source,
		LocalDir: DATA_DIR,
	}

	flog.Infoln("Downloading CPE dictionnary")
	if err := dfs.Do(ctx); err != nil {
		return err
	}

	dict_compressed, err := os.Open(DATA_DIR + "/official-cpe-dictionary_v2.3.xml.gz")
	if err != nil {
		return err
	}

	dict, err := gzip.NewReader(dict_compressed)
	if err != nil {
		dict_compressed.Close()
		return err
	}

	flog.Infoln("Decoding xml dictionnary")
	list, err := cpedict.Decode(dict)
	dict_compressed.Close()
	dict.Close()
	if err != nil {
		return err
	}

	flog.Infoln("Inserting into the database")
	const batch_size = 128
	var batch []CPERecord
	var references_batch []ReferenceRecord
	for i, item := range list.Items {
		if i != 0 && i%batch_size == 0 {
			err := insertBatch(batch, references_batch, db, ctx)
			if err != nil {
				return err
			}
			batch = nil
			references_batch = nil
			if i/batch_size%50 == 0 {
				flog.Debug("Inserting batch: ", i/batch_size)
			}
		}

		record := CPERecord{
			Part:      item.Name.Part,
			Vendor:    item.Name.Vendor,
			Product:   item.Name.Product,
			Version:   item.Name.Version,
			Update:    item.Name.Update,
			Edition:   item.Name.Edition,
			SWEdition: item.Name.SWEdition,
			TargetSW:  item.Name.TargetSW,
			TargetHW:  item.Name.TargetHW,
			Other:     item.Name.Other,
			Language:  item.Name.Language,
			Title:     item.Title["en-US"],
			URI:       (wfn.Attributes)(item.Name).BindToFmtString(),
		}
		batch = append(batch, record)

		for _, ref := range item.References {
			rec := ReferenceRecord{
				URI:         record.URI,
				URL:         ref.URL,
				Description: ref.Desc,
			}
			references_batch = append(references_batch, rec)
		}
	}
	err = insertBatch(batch, references_batch, db, ctx)
	if err != nil {
		return err
	}

	flog.Infoln("CPE import over")
	return nil
}

func ImportCron(db *sql.DB, delay time.Duration) {
	for {
		if err := ImportDictionnary(db, context.Background()); err != nil {
			flog.Error(err)
		}
		time.Sleep(delay)
	}
}
