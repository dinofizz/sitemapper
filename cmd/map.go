package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"net/url"
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
	versionCmd.Flags().IntVarP(&limit, "limit", "l", 10, "Specify mode: synchronous, concurrent, limited")
	err := versionCmd.MarkFlagRequired("site")
	if err != nil {
		log.Fatalf(err.Error())
	}
}

var versionCmd = &cobra.Command{
	Use:   "map",
	Short: "Create a sitemap",
	RunE: func(cmd *cobra.Command, args []string) error {
		startUrl, err := url.Parse(strings.ToLower(site))
		if err != nil {
			return err
		}
		sm := internal.NewSiteMap()
		var v internal.CrawlEngine
		v = internal.NewConcurrentCrawlEngine(sm, depth, startUrl)
		if mode != "" {
			switch mode {
			case "synchronous":
				v = internal.NewSynchronousCrawlEngine(sm, depth, startUrl)
			case "concurrent":
				v = internal.NewConcurrentCrawlEngine(sm, depth, startUrl)
			case "limited":
				if limit <= 0 {
					return errors.New("invalid limit")
				}
				l := internal.NewLimiter(limit)
				v = internal.NewConcurrentLimitedCrawlEngine(sm, depth, startUrl, l)
			default:
				return errors.New("unsupported mode")
			}
		}

		mc := &internal.Crawler{V: v}
		log.Printf("Crawling %s with depth %d", site, depth)
		start := time.Now()
		mc.Run()
		end := time.Now()
		elapsed := end.Sub(start)
		sm.Print()
		fmt.Println("Elapsed milliseconds: ", elapsed.Milliseconds())
		return nil
	},
}
