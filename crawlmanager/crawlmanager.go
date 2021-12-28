package main

import (
	"github.com/google/uuid"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type crawlManager struct {
	jm   *JobManager
	cass *cass
	nm   *natsManager
}

type crawl struct {
	ID        string
	SitemapID string
	URL       string
	Depth     int
}

type start struct {
	SitemapID string
	URL       string
	MaxDepth  int
}

type result struct {
	URL   string
	Links []string
}

type results struct {
	CrawlId string
	Results *[]result
}

func main() {
	log.Println("Starting Crawl Manager")
	rand.Seed(time.Now().Unix())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)

	jm := NewJobManager()
	cass := NewCass()
	cm := &crawlManager{jm: jm, cass: cass}
	nm := NewNATSManager(cm.HandleStartMessage, cm.HandleCrawlMessage, cm.HandleResultsMessage)
	cm.nm = nm

	wg := sync.WaitGroup{}
	wg.Add(1)

	nm.Start()
	defer nm.Stop()

	go func() {
		s := <-sigs
		log.Printf("Received signal %s\n", s.String())
		nm.Stop()
		wg.Done()
	}()

	createReadyFile()
	wg.Wait()
}

func createReadyFile() {
	emptyFile, err := os.Create("/ready/ready.txt")
	if err != nil {
		panic(err)
	}
	log.Println("Created ready.txt")
	emptyFile.Close()
}

func (cm *crawlManager) HandleStartMessage(s *start) {
	log.Printf("[Start] %v", *s)
	err := cm.cass.WriteSitemap(s.SitemapID, s.URL, s.MaxDepth)
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

	err = cm.nm.SendCrawlMessage(crawlID, sitemapID, s.URL, 1)
	if err != nil {
		log.Print(err)
		return
	}
}

func (cm *crawlManager) HandleCrawlMessage(c *crawl) {
	log.Printf("[Crawl] %v", *c)
	crawlID, err := uuid.Parse(c.ID)
	if err != nil {
		log.Print(err)
		return
	}

	sitemapID, err := uuid.Parse(c.SitemapID)
	if err != nil {
		log.Print(err)
		return
	}

	md, err := cm.cass.GetMaxDepthForSitemapID(sitemapID)
	if err != nil {
		log.Print(err)
		return
	}

	if c.Depth <= md {

		exists, err := cm.cass.URLExistsForSitemapID(sitemapID, c.URL)
		if err != nil {
			log.Print(err)
			return
		}

		if exists {
			log.Printf("URL %s already exists for sitemap ID %s", c.URL, sitemapID)
			return
		}

		err = cm.cass.WriteCrawl(crawlID, sitemapID, c.URL, c.Depth, md, "PENDING")
		if err != nil {
			log.Print(err)
			return
		}

		err = cm.jm.CreateJob(crawlID, c.URL)
		if err != nil {
			if strings.Contains(err.Error(), "exceeded quota") {
				log.Printf("Too many jobs, re-flighting message for crawl ID: %s\n", c.ID)
				time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
				err = cm.nm.SendCrawlMessage(crawlID, sitemapID, c.URL, c.Depth)
				if err != nil {
					log.Print(err)
				}
			} else {
				log.Print(err)
			}
			return
		}
		err = cm.cass.UpdateStatus(crawlID, sitemapID, "CREATED")
		if err != nil {
			log.Print(err)
			return
		}
	}
}

func (cm *crawlManager) HandleResultsMessage(r *results) {
	log.Printf("[Results] Crawl ID: %s", r.CrawlId)

	crawlID, err := uuid.Parse(r.CrawlId)
	if err != nil {
		log.Print(err)
		return
	}
	cj, err := cm.cass.GetSitemapIDForCrawlID(crawlID)
	if err != nil {
		log.Print(err)
		return
	}

	for _, rs := range *r.Results {
		err := cm.cass.WriteResults(cj.sitemapID, cj.crawlID, rs.URL, rs.Links)
		if err != nil {
			log.Print(err)
			continue
		}
		for _, link := range rs.Links {
			nextDepth := cj.depth + 1

			if nextDepth <= cj.maxDepth {
				newCrawlID, err := uuid.NewUUID()
				if err != nil {
					log.Print(err)
					return
				}

				err = cm.nm.SendCrawlMessage(newCrawlID, cj.sitemapID, link, nextDepth)
				if err != nil {
					log.Print(err)
					return
				}
			}
		}
	}

}
