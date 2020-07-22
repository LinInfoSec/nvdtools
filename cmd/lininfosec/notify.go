package main

import (
	"log"
	"database/sql"
	"context"
	"sync"
	
	_ "github.com/go-sql-driver/mysql"

	"github.com/pkg/errors"
	"github.com/facebookincubator/flog"
	"github.com/facebookincubator/nvdtools/providers/nvd"
	"github.com/facebookincubator/nvdtools/cvefeed"
	nvdfeed "github.com/facebookincubator/nvdtools/cvefeed/nvd"
	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/facebookincubator/nvdtools/vulndb/sqlutil"
	"github.com/facebookincubator/nvdtools/wfn"
)

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
	s []string
	err error
}


func getStacks(db *sql.DB, ctx context.Context, chunkSize int) <-chan pair {
	ch := make(chan pair, chunkSize)

	q := sqlutil.Select("uid").
		From("monitored_stacks")

	query:= q.String()

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		ch <- pair{
			s: nil,
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
					s: nil,
					err: errors.Wrap(err,"StackChannel scan failed"),
				}
				return
			}

			stack, err := getSoftwareStack(db,ctx, uid)
			if err != nil {
				ch <- pair{
					s: nil,
					err: errors.Wrap(err,"Failed to get a stackscan failed"),
				}
				return
			}
			stack = append([]string{uid}, stack...)
			ch <- pair{
				s: stack,
				err: nil,
			}

		}
	}()

	return ch
}

type VulnerableStack struct {
	Stack string                             `json:"stack"`
	cveId string
	CVE   *schema.NVDCVEFeedJSON10DefCVEItem  `json:"cve"`
}

func processAll(in <-chan []string, out chan<- VulnerableStack, caches map[string]*cvefeed.Cache) {
	const cpesAt = 1
	for rec := range in {
		if len(rec) == 1 {
			flog.Errorf("Empty software stack", len(rec))
			continue
		} else if len(rec) == 0{
			flog.Errorf("Unnamed software stack", len(rec))
			continue
		}

		cpeList := rec[cpesAt:]
		cpes := make([]*wfn.Attributes, 0, len(cpeList))
		for _, uri := range cpeList {
			attr, err := wfn.Parse(uri)
			if err != nil {
				flog.Errorf("couldn't parse uri %q: %v", uri, err)
				continue
			}
			cpes = append(cpes, attr)
		}

		for _, cache := range caches {
			for _, matches := range cache.Get(cpes) {
				ml := len(matches.CPEs)
				matchingCPEs := make([]string, ml)
				for i, attr := range matches.CPEs {
					if attr == nil {
						flog.Errorf("%s matches nil CPE", matches.CVE.ID())
						continue
					}
					matchingCPEs[i] = (*wfn.Attributes)(attr).BindToURI()
				}
				rec2 := make([]string, len(rec))
				copy(rec2, rec)
				cvss := matches.CVE.CVSSv3BaseScore()
				if cvss == 0 {
					cvss = matches.CVE.CVSSv2BaseScore()
				}
				
				switch v := matches.CVE.(type) {
					case *nvdfeed.Vuln:
						vuln := VulnerableStack {
							Stack: rec[0],
							cveId: v.ID(),
							CVE: v.Schema(),
						}
						out <- vuln
					default:
						flog.Errorf("Bad vuln type: %#v\n",v)
					
				}


			}
		}
	}
}

func filterNotifications(db *sql.DB, ctx context.Context, vulns []VulnerableStack) ([]VulnerableStack, error) {
	filtered := make([]VulnerableStack,0)

	for _,s := range vulns {
		tx, err := db.Begin()
		if err != nil {
			return nil, errors.Wrap(err, "unable to begin filter transation")
		}

		rows, err := tx.QueryContext(ctx,"SELECT COUNT(*) FROM cves_notified WHERE cve_id=? AND stack_uid=?",s.cveId,s.Stack)
		if err != nil {
			rows.Close()
			tx.Rollback()
			return nil, errors.Wrap(err, "Failed to check if the vulnerability has already been notified")
		}

		var count int
		rows.Next()
		err = rows.Scan(&count)
		rows.Close()
		if err != nil {
			tx.Rollback()
			return nil, errors.Wrap(err, "Failed to check the notification count")
		}

		if count != 0 {
			tx.Commit()
			continue
		} else {
			_, err := tx.ExecContext(ctx,"INSERT INTO cves_notified (cve_id,stack_uid) VALUES (?,?)",s.cveId,s.Stack)
			if err != nil {
				tx.Rollback()
				return nil, errors.Wrap(err, "Failed to store the notification")
			}
			filtered = append(filtered,s)
		}

		err = tx.Commit()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to commit notification transaction")
		}
	}

	return filtered, nil
}

func Notifications(db *sql.DB) ([]VulnerableStack, error) {

	var feed nvd.CVE
	feed.Set("cve-1.1.json.gz")
	//source := nvd.NewSourceConfig()

	//dfs := nvd.Sync{
	//	Feeds:    []nvd.Syncer{feed},
	//	Source:   source,
	//	LocalDir: DATA_DIR,
	//}

	ctx := context.Background()

	log.Println("Loading latest CVEs")
	//if err := dfs.Do(ctx); err != nil {
	//	flog.Errorf("%#v",err)
	//}

	log.Println("Parsing recent CVEs dictionary")
	recentFile := DATA_DIR + "/nvdcve-1.1-recent.json.gz"
	recent, err := cvefeed.LoadJSONDictionary([]string{recentFile}...)
	if err != nil {
		return nil, errors.Wrap(err,"failed to load recent cves")
	}
	
	log.Println("Parsing modified CVEs dictionary")
	modifiedFile := DATA_DIR + "/nvdcve-1.1-modified.json.gz"
	modified, err := cvefeed.LoadJSONDictionary([]string{modifiedFile}...)
	if err != nil {
		return nil, errors.Wrap(err,"failed to load modified cves")
	}

	log.Println("Creating CVE caches")
	caches := map[string]*cvefeed.Cache{}
	caches["recent"] = cvefeed.NewCache(recent).SetRequireVersion(true)
	caches["modified"] = cvefeed.NewCache(modified).SetRequireVersion(true)

	pairsChs := getStacks(db,ctx,4)
	stacksCh:= make(chan []string,4)
	vCh := make(chan VulnerableStack,4)
	

	// spawn processing goroutines
	var procWG sync.WaitGroup
	procWG.Add(4)
	for i := 0; i < 4; i++ {
		go func() {
			processAll(stacksCh, vCh, caches)
			procWG.Done()
		}()
	}



	for pair := range pairsChs {
		if pair.err != nil {
			close(stacksCh)
			procWG.Wait()
			close(vCh)
			return nil, pair.err
		}
	
		stacksCh <- pair.s
	}

	res := make([]VulnerableStack,0)
	go func() {
		for v := range vCh {
			res = append(res,v)
		}
	}()

	close(stacksCh)
	procWG.Wait()
	close(vCh)
	log.Println("Found ",len(res), "vulnerabilities")
	res_filtered, err := filterNotifications(db,ctx,res)
	if err != nil {
		return nil, err
	}
	log.Println("Found ",len(res_filtered), "vulnerabilities missing notifications")

	return res_filtered, nil
}

