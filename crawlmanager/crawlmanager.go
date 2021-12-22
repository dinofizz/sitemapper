package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type CrawlManager struct {
	jm *JobManager
}

type crawl struct {
	ID  string
	URL string
}

type results struct {
	CrawlId string
	Results *map[string][]string
}

func main() {
	log.Println("Starting Crawl Manager")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)

	jm := NewJobManager()
	cm := &CrawlManager{jm: jm}
	nm := NewNATSManager(cm.HandleCrawlMessage, cm.HandleResultsMessage)

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

	wg.Wait()
}

func (cm *CrawlManager) HandleCrawlMessage(c *crawl) {
	log.Printf("Crawl ID: %s, URL: %s", c.ID, c.URL)
	cm.jm.CreateJob(c.ID, c.URL)
}
func (cm *CrawlManager) HandleResultsMessage(r *results) {
	log.Printf("Crawl ID: %s, results: %v", r.CrawlId, r.Results)
}
