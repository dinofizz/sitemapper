package cmd

import (
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
var synchronous bool

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().IntVarP(&depth, "depth", "d", 1, "Specify crawl depth")
	versionCmd.Flags().StringVarP(&site, "site", "s", "", "Site to crawl, including http scheme")
	versionCmd.Flags().BoolVar(&synchronous, "synchronous", false, "Use concurrency (default is true)")
	versionCmd.MarkFlagRequired("site")
}

var versionCmd = &cobra.Command{
	Use:   "map",
	Short: "Create a sitemap",
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		log.Printf("Crawling %s with depth %d", site, depth)
		u, err := url.Parse(strings.ToLower(site))
		if err != nil {
			return err
		}
		sm := internal.NewSiteMap()
		c := internal.NewCrawler(sm, synchronous)
		c.Visit(u, u, 0, depth)
		if synchronous == false {
			c.WG.Wait()
		}
		end := time.Now()
		elapsed := end.Sub(start)
		sm.Print()
		fmt.Println(elapsed.Milliseconds())
		return nil
	},
}
