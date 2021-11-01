package cmd

import (
	"github.com/spf13/cobra"
	"log"
	"net/url"
	"sitemapper/internal"
	"strings"
)

var depth int
var site string

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().IntVarP(&depth, "depth", "d", 1, "Specify crawl depth")
	versionCmd.Flags().StringVarP(&site, "site", "s", "", "Site to crawl, including http scheme")
	versionCmd.MarkFlagRequired("site")
}

var versionCmd = &cobra.Command{
	Use:   "map",
	Short: "Create a sitemap",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Printf("Crawling %s with depth %d", site, depth)
		u, err := url.Parse(strings.ToLower(site))
		if err != nil {
			return err
		}
		sm := internal.NewSiteMap()
		c := internal.NewCrawler(sm)
		c.Visit(u, u, 0, depth)
		sm.Print()
		return nil
	},
}
