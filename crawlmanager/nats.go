package main

import (
	"github.com/nats-io/nats.go"
	"log"
	"os"
)

type CrawlMessageHandlerFunc func(c *crawl)
type ResultsMessageHandlerFunc func(c *results)

type NATSManager struct {
	CrawlMessageHandler   CrawlMessageHandlerFunc
	ResultsMessageHandler ResultsMessageHandlerFunc
	crawlSubject          string
	resultsSubject        string
	conn                  *nats.Conn
	server                string
	encodedConn           *nats.EncodedConn
}

func NewNATSManager(c CrawlMessageHandlerFunc, r ResultsMessageHandlerFunc) *NATSManager {
	n := &NATSManager{
		CrawlMessageHandler:   c,
		ResultsMessageHandler: r,
	}
	n.server = os.Getenv("NATS_SERVER")
	if n.server == "" {
		log.Fatalf("Unable to find NATS_SERVER in env vars")
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

func (n *NATSManager) Start() {
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

	if _, err := ec.Subscribe(n.crawlSubject, n.CrawlMessageHandler); err != nil {
		log.Fatal(err)
	}
	log.Printf("Subscribed to %s\n", n.crawlSubject)
	if _, err := ec.Subscribe(n.resultsSubject, n.ResultsMessageHandler); err != nil {
		log.Fatal(err)
	}
	log.Printf("Subscribed to %s\n", n.resultsSubject)
}

func (n *NATSManager) Stop() {
	n.encodedConn.Close()
	n.conn.Close()
}
