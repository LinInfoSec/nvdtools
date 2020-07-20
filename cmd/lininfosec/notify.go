package main

import (
	"log"
	"database/sql"
	"context"
	
	_ "github.com/go-sql-driver/mysql"

	"github.com/pkg/errors"
	"github.com/facebookincubator/nvdtools/providers/nvd"
	"github.com/facebookincubator/nvdtools/cvefeed"
	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/facebookincubator/nvdtools/vulndb/sqlutil"
)

type VulnerableCpe struct {
	Stack string                             `json:"stack"`
	CVE   schema.NVDCVEFeedJSON10DefCVEItem  `json:"cve"`
}

func getSoftwareStack(db *sql.DB, ctx context.Context, uid string) ([]string, error) {
	q := sqlutil.Select("cpe_uri").
		From("cpe_monitored").
		Where(sqlutil.Cond().Equal("stack_uid",uid))

	query, args := q.String(), q.QueryArgs()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil,errors.Wrap(err,"StackQuery failed")
	}
	defer rows.Close()
	
	var stack []string
	for rows.Next() {
		uri := ""
		err = rows.Scan(&uri)
		if err != nil {
			return nil,errors.Wrap(err, "Stack scan")
		}

		stack = append(stack,uri)
	}


	return stack,nil
}

type pair struct {
	s string
	err error
}


func getStacksUids(db *sql.DB, ctx context.Context, chunkSize int) <-chan pair {
	ch := make(chan pair, chunkSize)

	q := sqlutil.Select("uid").
		From("monitored_stacks")

	query:= q.String()

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		ch <- pair{
			s: "",
			err: errors.Wrap(err, "Error getting stacks uids"),
		}
		return ch
	}

	go func() {
		defer rows.Close()
		defer close(ch)
		for rows.Next() {
			uid := ""
			err := rows.Scan(&uid)
			if err != nil {
				ch <- pair{
					s: "",
					err: errors.Wrap(err,"StackChannel scan failed"),
				}
				return
			}
			ch <- pair{
					s: uid,
					err: nil,
				}

		}
	}()

	return ch
}

func Notifications(db *sql.DB) ([]VulnerableCpe, error) {
	log.Println("Notifications")

	var feed nvd.CVE
	feed.Set("cve-1.1.json.gz")
	source := nvd.NewSourceConfig()

	dfs := nvd.Sync{
		Feeds:    []nvd.Syncer{feed},
		Source:   source,
		LocalDir: DATA_DIR,
	}

	ctx := context.Background()

	log.Println("Loading latest CVEs")
	if err := dfs.Do(ctx); err != nil {
		log.Fatal(err)
	}

	log.Println("Parsing recent CVEs dictionary")
	recentFile := DATA_DIR + "/nvdcve-1.1-recent.json.gz"
	recent, err := cvefeed.LoadJSONDictionary([]string{recentFile}...)
	if err != nil {
		log.Fatal("failed to load recent cves",err)
	}

	log.Println("Parsing modified CVEs dictionary")
	modifiedFile := DATA_DIR + "/nvdcve-1.1-modified.json.gz"
	modified, err := cvefeed.LoadJSONDictionary([]string{modifiedFile}...)
	if err != nil {
		log.Fatal("failed to load recent cves",err)
	}
	caches := map[string]*cvefeed.Cache{}
	caches["recent"] = cvefeed.NewCache(recent).SetRequireVersion(true).SetMaxSize(50)
	caches["modified"] = cvefeed.NewCache(modified).SetRequireVersion(true).SetMaxSize(50)

	stacksCh := getStacksUids(db,ctx,4)

	for tmp := range stacksCh {
		s, err := tmp.s, tmp.err
		if err != nil {
			return nil,err
		}
		sStack, err := getSoftwareStack(db,ctx,s)
		if err != nil {
			return nil, err
		}
		log.Println(sStack)
	}
	
	return make([]VulnerableCpe,0),nil
}
