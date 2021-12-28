package main

import (
	sitemap "github.com/dinofizz/sitemapper/sitemapper/internal"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	log.Println("Starting Crawl Manager")
	rand.Seed(time.Now().Unix())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)

	jm := sitemap.NewJobManager()
	cass := sitemap.NewCass()
	cm := &sitemap.CrawlManager{JobManager: jm, CassDB: cass}
	nm := sitemap.NewNATSManager()
	cm.NatsManager = nm

	wg := sync.WaitGroup{}
	wg.Add(1)

	nm.SubscribeResultsSubject(cm.HandleResultsMessage)
	nm.SubscribeStartSubject(cm.HandleStartMessage)
	nm.SubscribeCrawlSubject(cm.HandleCrawlMessage)

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
