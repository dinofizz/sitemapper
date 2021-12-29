package main

import (
	"errors"
	"fmt"
	"github.com/dinofizz/sitemapper/sitemapper/internal"
	"github.com/spf13/cobra"
	"log"
	"os"
	"strings"
	"time"
)

var depth int
var site string
var mode string
var limit int

func init() {
	rootCmd.Flags().IntVarP(&depth, "depth", "d", 1, "Specify crawl depth")
	rootCmd.Flags().StringVarP(&site, "site", "s", "", "Site to crawl, including http scheme")
	rootCmd.Flags().StringVarP(&mode, "mode", "m", "concurrent", "Specify mode: synchronous, concurrent, limited")
	rootCmd.Flags().IntVarP(&limit, "limit", "l", 10, "Specify max concurrent crawl tasks for limited mode")
	err := rootCmd.MarkFlagRequired("site")
	if err != nil {
		log.Fatalf(err.Error())
	}
}

var rootCmd = &cobra.Command{
	Use:   "sm",
	Short: "Crawls from a start URL and writes a JSON based sitemap to stdout",
	RunE: func(cmd *cobra.Command, args []string) error {
		startUrl := strings.ToLower(site)
		sm := sitemap.NewSiteMap()
		var c sitemap.CrawlEngine
		c = sitemap.NewConcurrentCrawlEngine(sm, depth, startUrl)
		if mode != "" {
			switch mode {
			case "concurrent": // default mode if none specified
			case "synchronous":
				c = sitemap.NewSynchronousCrawlEngine(sm, depth, startUrl)
			case "limited":
				if limit <= 0 {
					return errors.New("invalid limit")
				}
				l := sitemap.NewLimiter(limit)
				c = sitemap.NewConcurrentLimitedCrawlEngine(sm, depth, startUrl, l)
			default:
				return errors.New("unsupported mode")
			}
		}

		log.Printf("Crawling %s with depth %d", site, depth)
		start := time.Now()
		c.Run()
		end := time.Now()
		elapsed := end.Sub(start)
		log.Println("Elapsed milliseconds: ", elapsed.Milliseconds())
		_, err := sm.WriteTo(os.Stdout)
		return err
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
