package sitemap

import (
	"github.com/google/uuid"
	"log"
	"math/rand"
	"strings"
	"time"
)

type CrawlManager struct {
	JobManager  *JobManager
	CassDB      *Cass
	NatsManager *NATS
}

func (cm *CrawlManager) HandleStartMessage(s *StartMessage) {
	log.Printf("[Start] %v", *s)
	err := cm.CassDB.WriteSitemap(s.SitemapID, s.URL, s.MaxDepth)
	if err != nil {
		log.Print(err)
		return
	}
	sitemapID, err := uuid.Parse(s.SitemapID)
	if err != nil {
		log.Print(err)
		return
	}

	crawlID, err := uuid.NewUUID()
	if err != nil {
		log.Print(err)
		return
	}

	err = cm.NatsManager.SendCrawlMessage(crawlID, sitemapID, s.URL, 1)
	if err != nil {
		log.Print(err)
		return
	}
}

func (cm *CrawlManager) HandleCrawlMessage(c *CrawlMessage) {
	log.Printf("[Crawl] %v", *c)
	crawlID, err := uuid.Parse(c.CrawlID)
	if err != nil {
		log.Print(err)
		return
	}

	sitemapID, err := uuid.Parse(c.SitemapID)
	if err != nil {
		log.Print(err)
		return
	}

	md, err := cm.CassDB.GetMaxDepthForSitemapID(sitemapID)
	if err != nil {
		log.Print(err)
		return
	}

	if c.CurrentDepth <= md {

		exists, err := cm.CassDB.URLExistsForSitemapID(sitemapID, c.URL)
		if err != nil {
			log.Print(err)
			return
		}

		if exists {
			log.Printf("URL %s already exists for sitemap ID %s", c.URL, sitemapID)
			return
		}

		err = cm.CassDB.WriteCrawl(crawlID, sitemapID, c.URL, c.CurrentDepth, md, "PENDING")
		if err != nil {
			log.Print(err)
			return
		}

		err = cm.JobManager.CreateJob(crawlID, c.URL)
		if err != nil {
			if strings.Contains(err.Error(), "exceeded quota") {
				log.Printf("Too many jobs, re-flighting message for crawl ID: %s\n", c.CrawlID)
				time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
				err = cm.NatsManager.SendCrawlMessage(crawlID, sitemapID, c.URL, c.CurrentDepth)
				if err != nil {
					log.Print(err)
				}
			} else {
				log.Print(err)
			}
			return
		}
		err = cm.CassDB.UpdateStatus(crawlID, sitemapID, "CREATED")
		if err != nil {
			log.Print(err)
			return
		}
	}
}

func (cm *CrawlManager) HandleResultsMessage(r *ResultsMessage) {
	log.Printf("[Results] Crawl ID: %s", r.CrawlId)

	crawlID, err := uuid.Parse(r.CrawlId)
	if err != nil {
		log.Print(err)
		return
	}
	cj, err := cm.CassDB.GetSitemapIDForCrawlID(crawlID)
	if err != nil {
		log.Print(err)
		return
	}

	for _, rs := range r.Results {
		err := cm.CassDB.WriteResults(cj.SitemapID, cj.CrawlID, rs.URL, rs.Links)
		if err != nil {
			log.Print(err)
			continue
		}
		for _, link := range rs.Links {
			nextDepth := cj.Depth + 1

			if nextDepth <= cj.MaxDepth {
				newCrawlID, err := uuid.NewUUID()
				if err != nil {
					log.Print(err)
					return
				}

				err = cm.NatsManager.SendCrawlMessage(newCrawlID, cj.SitemapID, link, nextDepth)
				if err != nil {
					log.Print(err)
					return
				}
			}
		}
		err = cm.CassDB.UpdateStatus(crawlID, cj.SitemapID, "COMPLETE")
		if err != nil {
			log.Print(err)
			return
		}
	}

}
