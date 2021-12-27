package sitemap

import (
	"bytes"
	"encoding/json"
	"github.com/nats-io/nats.go"
	"log"
	"os"
)

type NATSSender struct {
	server  string
	subject string
}

func NewNATSSender() *NATSSender {
	srv := os.Getenv("NATS_SERVER")
	if srv == "" {
		panic("Unable to find NATS_SERVER in env vars")
	}
	sub := os.Getenv("NATS_RESULTS_SUBJECT")
	if sub == "" {
		panic("Unable to find NATS_RESULTS_SUBJECT in env vars")
	}
	return &NATSSender{server: srv, subject: sub}
}

func (nw *NATSSender) SendMessage(crawlId string, b bytes.Buffer) error {
	type results []struct {
		URL   string
		Links []string
	}
	var rs results
	err := json.Unmarshal(b.Bytes(), &rs)
	if err != nil {
		return err
	}
	nc, err := nats.Connect(nw.server)
	if err != nil {
		return err
	}
	defer nc.Close()
	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		log.Fatal(err)
	}
	defer ec.Close()
	type resultsMessage struct {
		CrawlId string
		Results results
	}
	if err = ec.Publish(nw.subject, &resultsMessage{CrawlId: crawlId, Results: rs}); err != nil {
		return err
	}
	return nil
}
