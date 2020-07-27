package main

import (
	"context"
	"os"
	"log"
	"time"

	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/facebookincubator/flog"
	"github.com/facebookincubator/nvdtools/cpedict"
	"github.com/facebookincubator/nvdtools/wfn"
	"github.com/facebookincubator/nvdtools/vulndb/sqlutil"
)

// CPERecord represents a db record of the `cpe_dict` table.
type CPERecord struct {
	Part       string  `sql:"Part"`
	Vendor     string  `sql:"Vendor"`
	Product    string  `sql:"Product"`
	Version    string  `sql:"Version"`
	Update     string  `sql:"UpdateCl"`
	Edition    string  `sql:"Edition"`
	SWEdition  string  `sql:"SWEdition"`
	TargetSW   string  `sql:"TargetSW"`
	TargetHW   string  `sql:"TargetHW"`
	Other      string  `sql:"Other"`
	Language   string  `sql:"Language"`
	Title      string  `sql:"title"`
	URI        string  `sql:"URI"`
}

// ReferenceRecord represents a db record of the `cpe_references` table.
type ReferenceRecord struct {
	URI          string `sql:"cpe_uri"`
	URL          string `sql:"url"`
	Description  string `sql:"description"`
}

// Insert a batch of CPEs into the database
func insertBatch(batch []CPERecord,references []ReferenceRecord,db *sql.DB, ctx context.Context) error {
	if len(batch) != 0 {
		records := sqlutil.NewRecords(batch)
		q := sqlutil.Replace().
			Into("cpe_dict").
			Fields(records.Fields()...).
			Values(records...)

		query, args  := q.String(), q.QueryArgs()
		_, err := db.ExecContext(ctx, query, args...)
		if err != nil {
			log.Printf("%#v",err)
			log.Printf("%#v",batch)
			return errors.Wrap(err, "Cannot insert CPE batch")
		}
	}

	if len(references)  != 0 {
		records := sqlutil.NewRecords(references)
		q := sqlutil.Replace().
			Into("cpe_references").
			Fields(records.Fields()...).
			Values(records...)

		query, args  := q.String(), q.QueryArgs()
		_, err := db.ExecContext(ctx, query, args...)
		if err != nil {
			log.Printf("%#v",err)
			log.Printf("%#v",batch)
			return errors.Wrap(err, "Cannot insert CPE references")
		}
	}

	return nil
}

func ImportDictionnary(db *sql.DB,ctx context.Context, dictionnaryPath string) error {

	//TODO redownload the dictionnary automatically
	dict, err := os.Open(dictionnaryPath)
	if err != nil {
		dict.Close()
		return err
	}

	log.Println("CPE dictionnary import launched")
	log.Println("Decoding xml dictionnary")
	list, err := cpedict.Decode(dict)
	dict.Close()
	if err != nil {
		return err
	}

	log.Println("Inserting into the database")
	const batch_size =  128
	var batch []CPERecord
	var references_batch []ReferenceRecord;
	for i , item := range list.Items {
		if  i != 0 && i % batch_size == 0 {
			err:= insertBatch(batch,references_batch, db, ctx)
			if err != nil {
				return err
			}
			batch = nil
			references_batch = nil
			if i / batch_size % 50 == 0 {
				log.Println("Inserting batch", i/batch_size)
			}
		}
		

		record := CPERecord {
			Part:        item.Name.Part,
			Vendor:      item.Name.Vendor,
			Product:     item.Name.Product,
			Version:     item.Name.Version,
			Update:      item.Name.Update,
			Edition:     item.Name.Edition,
			SWEdition:   item.Name.SWEdition,
			TargetSW:    item.Name.TargetSW,
			TargetHW:    item.Name.TargetHW,
			Other:       item.Name.Other,
			Language:    item.Name.Language,
			Title:       item.Title["en-US"],
			URI:         (wfn.Attributes)(item.Name).BindToFmtString(),
		}
		batch = append(batch, record)

		for _, ref := range item.References {
			rec := ReferenceRecord {
				URI: record.URI,
				URL: ref.URL,
				Description: ref.Desc,
			}
			references_batch = append(references_batch, rec)
		}
	}
	err = insertBatch(batch,references_batch, db, ctx)
	if err != nil {
		return err
	}
	
	log.Println("CPE import over")
	return nil
}

func ImportCron(db *sql.DB, delay time.Duration) {
	for {
		if err := ImportDictionnary(db,context.Background(),DATA_DIR + "/official-cpe-dictionary_v2.3.xml"); err != nil {
			flog.Error(err)
		}
		time.Sleep(delay)
	}
}
