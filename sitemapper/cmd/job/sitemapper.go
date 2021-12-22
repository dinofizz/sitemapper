package main

import (
	"bytes"
	"fmt"
	"github.com/dinofizz/sitemapper/sitemapper/internal"
	"github.com/spf13/cobra"
	"log"
	"os"
	"strings"
	"time"
)

var site string
var id string

func init() {
	rootCmd.Flags().StringVarP(&site, "site", "s", "", "Site to crawl, including http scheme")
	rootCmd.Flags().StringVar(&id, "id", "", "Crawl job identifier")
	err := rootCmd.MarkFlagRequired("site")
	if err != nil {
		log.Fatalf(err.Error())
	}
	err = rootCmd.MarkFlagRequired("id")
	if err != nil {
		log.Fatalf(err.Error())
	}
}

var rootCmd = &cobra.Command{
	Use:   "sitemapper",
	Short: "Crawls from a start URL and writes a JSON based sitemap to a NATS topic",
	RunE: func(cmd *cobra.Command, args []string) error {
		startUrl := strings.ToLower(site)
		sm := sitemap.NewSiteMap()
		c := sitemap.NewSynchronousCrawlEngine(sm, 1, startUrl)
		log.Printf("Crawling %s", site)
		start := time.Now()
		c.Run()
		end := time.Now()
		elapsed := end.Sub(start)
		log.Println("Elapsed milliseconds: ", elapsed.Milliseconds())
		var b bytes.Buffer
		_, err := sm.WriteTo(&b)
		if err != nil {
			return err
		}

		ns := sitemap.NewNATSSender()
		return ns.SendMessage(id, b)
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
