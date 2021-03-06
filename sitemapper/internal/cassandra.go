package sitemap

import (
	"github.com/NathanBak/easy-cass-go/pkg/easycass"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"log"
	"os"
)

type AstraDB struct {
	session *gocql.Session
}

func getAstraDBCreds() (string, string, string) {
	id := os.Getenv("ASTRA_CLIENT_ID")
	if id == "" {
		log.Fatalf("Unable to find ASTRA_CLIENT_ID in env vars")
	}
	secret := os.Getenv("ASTRA_CLIENT_SECRET")
	if secret == "" {
		log.Fatalf("Unable to find ASTRA_CLIENT_SECRET in env vars")
	}
	zip := os.Getenv("ASTRA_CLIENT_ZIP_PATH")
	if zip == "" {
		log.Fatalf("Unable to find ASTRA_CLIENT_ZIP in env vars")
	}
	return id, secret, zip
}

func NewAstraDB() *AstraDB {
	log.Println("Connecting to AstraDB")
	id, secret, zipPath := getAstraDBCreds()
	session, err := easycass.GetSession(id, secret, zipPath)
	session.Closed()
	if err != nil {
		log.Fatal(err)
	}
	log.Println(easycass.GetKeyspace(session))
	return &AstraDB{session: session}
}

func (c *AstraDB) HealthCheck() error {
	var count int
	err := c.session.Query("SELECT COUNT(*) FROM sitemaps").Scan(&count)
	if err != nil {
		return err
	}
	return nil
}

func (c *AstraDB) WriteCrawl(crawlID, sitemapID uuid.UUID, url string, depth, maxDepth int, status string) error {
	cUUID, err := gocql.ParseUUID(crawlID.String())
	if err != nil {
		return err
	}

	var s string
	err = c.session.Query("SELECT status FROM crawl_jobs WHERE crawl_id = ?", cUUID).Scan(&s)
	if err != nil {
		if !errors.Is(err, gocql.ErrNotFound) {
			return errors.Wrap(err, "Error checking if crawl job exists")
		}
	}
	if s == "PENDING" {
		log.Printf("Crawl ID %s already exists in PENDING state", cUUID)
		return nil
	}
	if s != "" {
		return errors.Errorf("Found crawl ID %s with existing status %s", crawlID, s)
	}

	smUUID, err := gocql.ParseUUID(sitemapID.String())
	if err != nil {
		return err
	}

	if err = c.session.Query(`INSERT INTO crawl_jobs (crawl_id, sitemap_id, url, depth, max_depth, status) VALUES (?, ?, ?, ? ,? ,?)`,
		cUUID, smUUID, url, depth, maxDepth, status).Exec(); err != nil {
		return errors.Wrap(err, "Unable to write job to DB")
	}
	return nil
}

func (c *AstraDB) UpdateStatus(crawlID, sitemapID uuid.UUID, status string) error {
	cUUID, err := gocql.ParseUUID(crawlID.String())
	if err != nil {
		return err
	}
	sUUID, err := gocql.ParseUUID(sitemapID.String())
	if err != nil {
		return err
	}
	if err = c.session.Query(`UPDATE crawl_jobs SET status = ? WHERE crawl_id = ? and sitemap_id = ?`,
		status, cUUID, sUUID).Exec(); err != nil {
		return errors.Wrapf(err, "Unable to update status for crawl ID %s, sitemap ID: %s", crawlID, sitemapID)
	}
	return nil
}

func (c *AstraDB) WriteSitemap(sitemapID string, url string, maxDepth int) error {
	smUUID, err := gocql.ParseUUID(sitemapID)
	if err != nil {
		return err
	}

	var count int
	err = c.session.Query("SELECT COUNT(*) FROM sitemaps WHERE sitemap_id = ?", smUUID).Scan(&count)
	if err != nil {
		return errors.Wrap(err, "Error checking if sitemap exists")
	}
	if count != 0 {
		return errors.Errorf("Sitemap ID %s already exists", smUUID)
	}

	if err = c.session.Query(`INSERT INTO sitemaps (sitemap_id, url, max_depth) VALUES (?, ?, ?)`,
		smUUID, url, maxDepth).Exec(); err != nil {
		return errors.Wrap(err, "Unable to write sitemap to DB")
	}
	return nil
}

type crawlJob struct {
	CrawlID   uuid.UUID
	SitemapID uuid.UUID
	Depth     int
	MaxDepth  int
}

func (c *AstraDB) GetSitemapIDForCrawlID(crawlID uuid.UUID) (*crawlJob, error) {
	cUUID, err := gocql.ParseUUID(crawlID.String())
	if err != nil {
		return nil, err
	}

	cj := &crawlJob{CrawlID: crawlID}
	var smUUID gocql.UUID

	err = c.session.Query("SELECT sitemap_id, depth, max_depth FROM crawl_jobs WHERE crawl_id = ?", cUUID).Scan(&smUUID, &(cj.Depth), &(cj.MaxDepth))
	if err != nil {
		return nil, errors.Wrapf(err, "Error checking for sitemap ID using crawl ID %s", crawlID)
	}

	sitemapID := uuid.MustParse(smUUID.String())
	cj.SitemapID = sitemapID

	return cj, nil
}

func (c *AstraDB) GetMaxDepthForSitemapID(sitemapID uuid.UUID) (int, error) {
	smUUID, err := gocql.ParseUUID(sitemapID.String())
	if err != nil {
		return -1, err
	}

	var maxDepth int
	err = c.session.Query("SELECT max_depth FROM sitemaps WHERE sitemap_id = ?", smUUID).Scan(&maxDepth)
	if err != nil {
		return -1, errors.Wrapf(err, "Error checking max_depth for sitemap ID %s", sitemapID.String())
	}

	return maxDepth, nil
}

func (c *AstraDB) URLExistsForSitemapID(sitemapID uuid.UUID, URL string) (bool, error) {
	smUUID, err := gocql.ParseUUID(sitemapID.String())
	if err != nil {
		return false, err
	}
	var count int

	err = c.session.Query("SELECT COUNT(*) FROM results_by_sitemap_id WHERE sitemap_id = ? AND url = ?", smUUID, URL).Scan(&count)
	if err != nil {
		return false, errors.Wrap(err, "Error checking if URL exists for sitemap ID")
	}

	if count == 0 {
		return false, nil
	}
	return true, nil
}

func (c *AstraDB) WriteResults(sitemapID, crawlID uuid.UUID, URL string, links []string) error {
	smUUID, err := gocql.ParseUUID(sitemapID.String())
	if err != nil {
		return err
	}

	cUUID, err := gocql.ParseUUID(crawlID.String())
	if err != nil {
		return err
	}

	if err = c.session.Query(`INSERT into results_by_sitemap_id ( sitemap_id, url, crawl_id, links) values (?, ?, ?, ?)`,
		smUUID, URL, cUUID, links).Exec(); err != nil {
		return errors.Wrap(err, "Unable to write results to DB")
	}
	return nil
}

type Details struct {
	SitemapID string
	URL       string
	MaxDepth  int
}

func (c *AstraDB) GetSitemapDetails(sitemapID uuid.UUID) (*Details, error) {
	smUUID, err := gocql.ParseUUID(sitemapID.String())
	if err != nil {
		return nil, err
	}

	var smDetails Details
	err = c.session.Query("SELECT url, max_depth FROM sitemaps where sitemap_id = ?", smUUID).Scan(&smDetails.URL, &smDetails.MaxDepth)
	if err != nil {
		return nil, err
	}

	smDetails.SitemapID = sitemapID.String()

	return &smDetails, nil
}

func (c *AstraDB) GetSitemapResults(sitemapID uuid.UUID) (*[]Result, error) {
	smUUID, err := gocql.ParseUUID(sitemapID.String())
	if err != nil {
		return nil, err
	}
	scanner := c.session.Query("SELECT url, links FROM results_by_sitemap_id WHERE sitemap_id = ?", smUUID).Iter().Scanner()

	var results []Result

	for scanner.Next() {
		var URL string
		var URLlinks []string

		err = scanner.Scan(&URL, &URLlinks)
		if err != nil {
			return nil, err
		}
		results = append(results, Result{URL: URL, Links: URLlinks})
	}

	if err = scanner.Err(); err != nil {
		return nil, err
	}

	return &results, nil
}
