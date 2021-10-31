package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"net/url"
	"os"
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
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Crawling %s with depth %d\n", site, depth)
		u, err := url.Parse(strings.ToLower(site))
		if err != nil {
			fmt.Printf("error parsing site URL %s\n", site)
			os.Exit(1)
		}
		sm := internal.NewSiteMap()
		c := internal.NewCrawler(sm)
		c.Visit(u, u, 0, depth)
		sm.Print()
	},
}
