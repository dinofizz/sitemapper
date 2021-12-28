package sitemap

import (
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"log"
	"os"
)

type CrawlMessageHandlerFunc func(c *CrawlMessage)
type ResultsMessageHandlerFunc func(c *ResultsMessage)
type StartMessageHandlerFunc func(c *StartMessage)

type CrawlMessage struct {
	ID        string
	SitemapID string
	URL       string
	Depth     int
}

type StartMessage struct {
	SitemapID string
	URL       string
	MaxDepth  int
}

type Result struct {
	URL   string
	Links []string
}

type ResultsMessage struct {
	CrawlId string
	Results []Result
}

type NATS struct {
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

func NewNATSManager() *NATS {
	n := &NATS{}
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
	conn, err := nats.Connect(n.server,
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
	n.conn = conn
	ec, err := nats.NewEncodedConn(n.conn, nats.JSON_ENCODER)
	if err != nil {
		log.Fatal(err)
	}
	n.encodedConn = ec
	log.Printf("Connected to NATS server %s\n", n.server)
	return n
}

func (n *NATS) SubscribeStartSubject(f StartMessageHandlerFunc) {
	subscribe(n.encodedConn, n.startSubject, f)
}

func (n *NATS) SubscribeCrawlSubject(f CrawlMessageHandlerFunc) {
	subscribe(n.encodedConn, n.crawlSubject, f)
}

func (n *NATS) SubscribeResultsSubject(f ResultsMessageHandlerFunc) {
	subscribe(n.encodedConn, n.resultsSubject, f)
}

func subscribe(ec *nats.EncodedConn, subject string, cb nats.Handler) {
	if _, err := ec.Subscribe(subject, cb); err != nil {
		log.Fatal(err)
	}
	log.Printf("Subscribed to %s\n", subject)
}

func (n *NATS) Stop() {
	n.encodedConn.Close()
	n.conn.Close()
}

func (n *NATS) SendCrawlMessage(crawlID, sitemapID uuid.UUID, URL string, depth int) error {
	if err := n.encodedConn.Publish(n.crawlSubject, &CrawlMessage{ID: crawlID.String(), URL: URL, Depth: depth, SitemapID: sitemapID.String()}); err != nil {
		return err
	}
	return nil
}
func (n *NATS) SendStartMessage(sitemapID uuid.UUID, URL string, maxDepth int) error {
	if err := n.encodedConn.Publish(n.startSubject, &StartMessage{URL: URL, MaxDepth: maxDepth, SitemapID: sitemapID.String()}); err != nil {
		return err
	}
	return nil
}
func (n *NATS) SendResultsMessage(crawlID uuid.UUID, results *[]Result) error {
	if err := n.encodedConn.Publish(n.resultsSubject, &ResultsMessage{CrawlId: crawlID.String(), Results: *results}); err != nil {
		log.Println(err)
		return err
	}
	return nil
}
