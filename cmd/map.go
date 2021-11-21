package cmd

import (
	"errors"
	"github.com/spf13/cobra"
	"log"
	"sitemapper/internal"
	"strings"
	"time"
)

var depth int
var site string
var mode string
var limit int

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().IntVarP(&depth, "depth", "d", 1, "Specify crawl depth")
	versionCmd.Flags().StringVarP(&site, "site", "s", "", "Site to crawl, including http scheme")
	versionCmd.Flags().StringVarP(&mode, "mode", "m", "concurrent", "Specify mode: synchronous, concurrent, limited")
	versionCmd.Flags().IntVarP(&limit, "limit", "l", 10, "Specify max concurrent crawl tasks for limited mode")
	err := versionCmd.MarkFlagRequired("site")
	if err != nil {
		log.Fatalf(err.Error())
	}
}

var versionCmd = &cobra.Command{
	Use:   "map",
	Short: "Create a sitemap",
	RunE: func(cmd *cobra.Command, args []string) error {
		startUrl := strings.ToLower(site)
		sm := internal.NewSiteMap()
		var c internal.CrawlEngine
		c = internal.NewConcurrentCrawlEngine(sm, depth, startUrl)
		if mode != "" {
			switch mode {
			case "synchronous":
				c = internal.NewSynchronousCrawlEngine(sm, depth, startUrl)
			case "concurrent":
				c = internal.NewConcurrentCrawlEngine(sm, depth, startUrl)
			case "limited":
				if limit <= 0 {
					return errors.New("invalid limit")
				}
				l := internal.NewLimiter(limit)
				c = internal.NewConcurrentLimitedCrawlEngine(sm, depth, startUrl, l)
			default:
				return errors.New("unsupported mode")
			}
		}

		crawler := &internal.Crawler{C: c}
		log.Printf("Crawling %s with depth %d", site, depth)
		start := time.Now()
		crawler.Run()
		end := time.Now()
		elapsed := end.Sub(start)
		log.Println("Elapsed milliseconds: ", elapsed.Milliseconds())
		sm.Dump()
		return nil
	},
}
