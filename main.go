package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"ecomcrawler/config"
	"ecomcrawler/exporter"
	"ecomcrawler/scraper"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {
	configFile := flag.String("config", "./site_configs.json", "Path to the JSON config file for sites.")
	outputDir := flag.String("output", "output_data", "Directory to save scraped data.")
	logLevel := flag.String("loglevel", "info", "Log level (debug, info, warn, error, fatal, panic).")
	flag.Parse()

	parsedLevel, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatalf("Invalid log level: %v", err)
	}
	log.SetLevel(parsedLevel)

	log.Info("EcomCrawler starting...")

	siteConfigs, err := config.LoadSiteConfigs(*configFile)
	if err != nil {
		log.Fatalf("Failed to load site configurations: %v", err)
	}
	if len(siteConfigs) == 0 {
		log.Fatal("No site configurations found. Exiting.")
	}

	log.Infof("Loaded %d site configurations.", len(siteConfigs))

	productChan := make(chan scraper.Product, 500)
	var wg sync.WaitGroup

	allProductsBySite := make(map[string][]scraper.Product)
	var mapMutex sync.Mutex

	for _, siteCfg := range siteConfigs {
		wg.Add(1)
		log.Infof("Dispatching crawler for site: %s", siteCfg.Name)
		go scraper.CrawlSite(siteCfg, productChan, &wg)
	}

	var collectorWg sync.WaitGroup
	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		for product := range productChan {
			// log.Infof("Received product: %s from %s", product.Name, product.SourceURL)
			for _, sc := range siteConfigs {
				for _, domain := range sc.AllowedDomains {
					if strings.Contains(product.SourceURL, domain) {
						mapMutex.Lock()
						foundSiteName := sc.Name
						allProductsBySite[foundSiteName] = append(allProductsBySite[foundSiteName], product)
						mapMutex.Unlock()
					}
				}
			}
		}
	}()

	wg.Wait()
	log.Info("All crawlers have finished.")

	close(productChan)
	log.Info("Product channel closed.")

	collectorWg.Wait()
	log.Info("Product collector has finished.")

	if err := os.MkdirAll(*outputDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create output directory %s: %v", *outputDir, err)
	}

	totalProductsScraped := 0
	for siteName, products := range allProductsBySite {

		if len(products) > 0 {
			wg.Add(1)

			go func(products []scraper.Product, siteName string) {
				defer wg.Done()

				log.Infof("Exporting %d products for site: %s", len(products), siteName)
				safeSiteName := strings.ReplaceAll(siteName, " ", "_")
				safeSiteName = strings.ReplaceAll(safeSiteName, "/", "_")
				safeSiteName = strings.ToLower(safeSiteName)

				err := exporter.ExportToJSON(products, *outputDir, safeSiteName)
				if err != nil {
					log.Errorf("Failed to export products for site %s: %v", siteName, err)
				} else {
					mapMutex.Lock()
					totalProductsScraped += len(products)
					mapMutex.Unlock()
				}
			}(products, siteName)

		}
	}
	wg.Wait()
	log.Infof("EcomCrawler finished. Total products scraped: %d. Output saved to directory: %s", totalProductsScraped, *outputDir)
	log.Infof("Check the '%s' directory for output JSON files.", *outputDir)
}

func getAbsPath(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Warnf("Could not get absolute path for %s: %v", path, err)
		return path
	}
	return absPath
}
