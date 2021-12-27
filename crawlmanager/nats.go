package main

import (
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"log"
	"os"
)

type CrawlMessageHandlerFunc func(c *crawl)
type ResultsMessageHandlerFunc func(c *results)
type StartMessageHandlerFunc func(c *start)

type natsManager struct {
	CrawlMessageHandler   CrawlMessageHandlerFunc
	ResultsMessageHandler ResultsMessageHandlerFunc
	StartMessageHandler   StartMessageHandlerFunc
	startSubject          string
	crawlSubject          string
	resultsSubject        string
	conn                  *nats.Conn
	server                string
	encodedConn           *nats.EncodedConn
}

func NewNATSManager(s StartMessageHandlerFunc, c CrawlMessageHandlerFunc, r ResultsMessageHandlerFunc) *natsManager {
	n := &natsManager{
		CrawlMessageHandler:   c,
		ResultsMessageHandler: r,
		StartMessageHandler:   s,
	}
	n.server = os.Getenv("NATS_SERVER")
	if n.server == "" {
		log.Fatalf("Unable to find NATS_SERVER in env vars")
	}
	n.startSubject = os.Getenv("NATS_START_SUBJECT")
	if n.startSubject == "" {
		log.Fatalf("Unable to find NATS_START_SUBJECT in env vars")
	}
	n.crawlSubject = os.Getenv("NATS_CRAWL_SUBJECT")
	if n.crawlSubject == "" {
		log.Fatalf("Unable to find NATS_CRAWL_SUBJECT in env vars")
	}
	n.resultsSubject = os.Getenv("NATS_RESULTS_SUBJECT")
	if n.resultsSubject == "" {
		log.Fatalf("Unable to find NATS_RESULTS_SUBJECT in env vars")
	}
	return n
}

func (n *natsManager) Start() {
	c, err := nats.Connect(n.server,
		nats.ErrorHandler(func(nc *nats.Conn, s *nats.Subscription, err error) {
			if s != nil {
				log.Printf("Async error in %q/%q: %v", s.Subject, s.Queue, err)
			} else {
				log.Printf("Async error outside subscription: %v", err)
			}
		}), nats.ClosedHandler(func(nc *nats.Conn) {
			log.Println("NATS connection closed")
		},
		))
	if err != nil {
		log.Fatal(err)
	}
	n.conn = c
	log.Printf("Connected to NATS server %s\n", n.server)
	ec, err := nats.NewEncodedConn(n.conn, nats.JSON_ENCODER)
	if err != nil {
		log.Fatal(err)
	}
	n.encodedConn = ec

	if _, err := ec.Subscribe(n.startSubject, n.StartMessageHandler); err != nil {
		log.Fatal(err)
	}
	log.Printf("Subscribed to %s\n", n.startSubject)
	if _, err := ec.Subscribe(n.crawlSubject, n.CrawlMessageHandler); err != nil {
		log.Fatal(err)
	}
	log.Printf("Subscribed to %s\n", n.crawlSubject)
	if _, err := ec.Subscribe(n.resultsSubject, n.ResultsMessageHandler); err != nil {
		log.Fatal(err)
	}
	log.Printf("Subscribed to %s\n", n.resultsSubject)
}

func (n *natsManager) Stop() {
	n.encodedConn.Close()
	n.conn.Close()
}

func (n *natsManager) SendCrawlMessage(crawlID, sitemapID uuid.UUID, URL string, depth int) error {
	if err := n.encodedConn.Publish(n.crawlSubject, &crawl{ID: crawlID.String(), URL: URL, Depth: depth, SitemapID: sitemapID.String()}); err != nil {
		return err
	}
	return nil
}
func (n *natsManager) SendStartMessage(sitemapID uuid.UUID, URL string, maxDepth int) error {
	if err := n.encodedConn.Publish(n.startSubject, &start{URL: URL, MaxDepth: maxDepth, SitemapID: sitemapID.String()}); err != nil {
		return err
	}
	return nil
}
