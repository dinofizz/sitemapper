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
	versionCmd.MarkFlagRequired("site")
}

var versionCmd = &cobra.Command{
	Use:   "map",
	Short: "Create a sitemap",
	RunE: func(cmd *cobra.Command, args []string) error {
		sync := true
		if mode != "" {
			switch mode {
			case "synchronous":
				sync = true
				limit = 0
			case "concurrent":
				sync = false
				limit = 0
			case "limited":
				if limit <= 0 {
					return errors.New("invalid limit")
				}
				sync = false
			default:
				return errors.New("unsupported mode")
			}
		}

		start := time.Now()
		log.Printf("Crawling %s with depth %d", site, depth)
		u, err := url.Parse(strings.ToLower(site))
		if err != nil {
			return err
		}
		sm := internal.NewSiteMap()
		c := internal.NewCrawler(sm, depth, sync, limit)
		c.Visit(u, u, 0)

		if mode == "concurrent" || mode == "limited" {
			c.WG.Wait()
		}



		end := time.Now()
		elapsed := end.Sub(start)
		sm.Print()
		fmt.Println(elapsed.Milliseconds())
		return nil
	},
}
