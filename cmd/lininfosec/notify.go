package main

import (
	"log"
	"database/sql"
	"net/http"
	"context"
	"sync"
	"time"
	"encoding/json"
	"bytes"
	
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

func getSoftwareConfiguration(db *sql.DB, ctx context.Context, uid string) ([]string, error) {
	q := sqlutil.Select("cpe_uri").
		From("cpe_monitored").
		Where(sqlutil.Cond().Equal("configuration_uid",uid))

	query, args := q.String(), q.QueryArgs()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil,errors.Wrap(err,"ConfigurationQuery failed")
	}
	defer rows.Close()
	
	var configuration []string
	for rows.Next() {
		uri := ""
		err = rows.Scan(&uri)
		if err != nil {
			return nil,errors.Wrap(err, "Configuration scan")
		}

		configuration = append(configuration,uri)
	}


	return configuration,nil
}

type pair struct {
	s []string
	err error
}


func getConfigurations(db *sql.DB, ctx context.Context, chunkSize int) <-chan pair {
	ch := make(chan pair, chunkSize)

	q := sqlutil.Select("uid").
		From("monitored_configurations")

	query:= q.String()

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		ch <- pair{
			s: nil,
			err: errors.Wrap(err, "Error getting configurations uids"),
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
					err: errors.Wrap(err,"ConfigurationChannel scan failed"),
				}
				return
			}

			configuration, err := getSoftwareConfiguration(db,ctx, uid)
			if err != nil {
				ch <- pair{
					s: nil,
					err: errors.Wrap(err,"Failed to get a configurationscan failed"),
				}
				return
			}
			configuration = append([]string{uid}, configuration...)
			ch <- pair{
				s: configuration,
				err: nil,
			}

		}
	}()

	return ch
}

type VulnerableConfiguration struct {
	Configuration string                             `json:"configuration"`
	cveId string
	CVE   *schema.NVDCVEFeedJSON10DefCVEItem  `json:"cve"`
}

func processAll(in <-chan []string, out chan<- VulnerableConfiguration, caches map[string]*cvefeed.Cache) {
	const cpesAt = 1
	for rec := range in {
		log.Printf("%#v",rec)
		if len(rec) == 1 {
			flog.Errorf("Empty software configuration", len(rec))
			continue
		} else if len(rec) == 0{
			flog.Errorf("Unnamed software configuration", len(rec))
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
						vuln := VulnerableConfiguration {
							Configuration: rec[0],
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

func filterNotifications(db *sql.DB, ctx context.Context, vulns []VulnerableConfiguration) ([]VulnerableConfiguration, error) {
	filtered := make([]VulnerableConfiguration,0)

	for _,s := range vulns {
		tx, err := db.Begin()
		if err != nil {
			return nil, errors.Wrap(err, "unable to begin filter transation")
		}

		rows, err := tx.QueryContext(ctx,"SELECT COUNT(*) FROM cves_notified WHERE cve_id=? AND configuration_uid=?",s.cveId,s.Configuration)
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
			_, err := tx.ExecContext(ctx,"INSERT INTO cves_notified (cve_id,configuration_uid) VALUES (?,?)",s.cveId,s.Configuration)
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

func Notifications(db *sql.DB) ([]VulnerableConfiguration, error) {
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
		flog.Errorf("%#v",err)
	}

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
	// TODO filter CVEs based on date to reduce the amount of filtering done by filterNotifications
	caches["recent"] = cvefeed.NewCache(recent).SetRequireVersion(true)
	caches["modified"] = cvefeed.NewCache(modified).SetRequireVersion(true)

	pairsChs := getConfigurations(db,ctx,4)
	configurationsCh:= make(chan []string,4)
	vCh := make(chan VulnerableConfiguration,4)
	

	// spawn processing goroutines
	var procWG sync.WaitGroup
	procWG.Add(4)
	for i := 0; i < 4; i++ {
		go func() {
			processAll(configurationsCh, vCh, caches)
			procWG.Done()
		}()
	}



	for pair := range pairsChs {
		if pair.err != nil {
			close(configurationsCh)
			procWG.Wait()
			close(vCh)
			return nil, pair.err
		}
	
		configurationsCh <- pair.s
	}

	res := make([]VulnerableConfiguration,0)
	go func() {
		for v := range vCh {
			res = append(res,v)
		}
	}()

	close(configurationsCh)
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

func NotificationCron(db *sql.DB, delay time.Duration) {
	if NOTIFICATION_ENDPOINT == "" {
		log.Fatal("No notification endpoint configured")
	}
	for {
		notifications, err := Notifications(db)
		if err != nil {
			flog.Errorf("Notification error: %#v",err)
		}

		serialized, err := json.Marshal(notifications)
		if err != nil {
			log.Fatal(err)
		}
		reader := bytes.NewReader(serialized)

		//TODO add way to authenticate
		res, err := http.Post(NOTIFICATION_ENDPOINT,"application/json",reader)
		if err != nil {
			flog.Errorf("Error sending notifications: %#v", err)
		} else if res.StatusCode < 200 || res.StatusCode >= 300 {
			flog.Errorf("Error sending notifications: %#v", res.Status)
		}
		time.Sleep(delay)
	}
}

