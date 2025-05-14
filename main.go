// ecomcrawler/cmd/ecomcrawler/main.go
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
	// Configure logrus
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel) // Default level, can be changed via flag
}

func main() {
	// Define command-line flags
	configFile := flag.String("config", "./site_configs.json", "Path to the JSON config file for sites.")
	outputDir := flag.String("output", "output_data", "Directory to save scraped data.")
	logLevel := flag.String("loglevel", "info", "Log level (debug, info, warn, error, fatal, panic).")
	flag.Parse()

	// Set log level from flag
	parsedLevel, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatalf("Invalid log level: %v", err)
	}
	log.SetLevel(parsedLevel)

	log.Info("EcomCrawler starting...")

	// Load site configurations
	siteConfigs, err := config.LoadSiteConfigs(*configFile)
	if err != nil {
		log.Fatalf("Failed to load site configurations: %v", err)
	}
	if len(siteConfigs) == 0 {
		log.Fatal("No site configurations found. Exiting.")
	}

	log.Infof("Loaded %d site configurations.", len(siteConfigs))

	// Create a channel to receive products from crawlers
	// Buffered channel to prevent crawlers from blocking if main goroutine is slow
	productChan := make(chan scraper.Product, 200*len(siteConfigs)) // Buffer size based on expected products per site

	// Create a WaitGroup to wait for all crawlers to finish
	var wg sync.WaitGroup

	// Map to store products per site
	allProductsBySite := make(map[string][]scraper.Product)
	var mapMutex sync.Mutex // Mutex to protect access to allProductsBySite

	// Start a goroutine for each site configuration
	for _, siteCfg := range siteConfigs {
		wg.Add(1) // Increment WaitGroup counter
		log.Infof("Dispatching crawler for site: %s", siteCfg.Name)
		go scraper.CrawlSite(siteCfg, productChan, &wg)
	}

	// Goroutine to collect products from the channel and close it once all crawlers are done
	var collectorWg sync.WaitGroup
	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		for product := range productChan {
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

	// Wait for all crawlers to complete their work
	wg.Wait()
	log.Info("All crawlers have finished.")

	// Close the product channel (no more products will be sent)
	close(productChan)
	log.Info("Product channel closed.")

	// Wait for the collector goroutine to finish processing all products from the channel
	collectorWg.Wait()
	log.Info("Product collector has finished.")

	// Ensure the output directory exists
	if err := os.MkdirAll(*outputDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create output directory %s: %v", *outputDir, err)
	}

	// Export products for each site
	totalProductsScraped := 0
	mapMutex.Lock() // Lock for reading allProductsBySite
	for siteName, products := range allProductsBySite {
		if len(products) > 0 {
			log.Infof("Exporting %d products for site: %s", len(products), siteName)
			// Sanitize siteName for use in filename
			safeSiteName := strings.ReplaceAll(siteName, " ", "_")
			safeSiteName = strings.ReplaceAll(safeSiteName, "/", "_")
			safeSiteName = strings.ToLower(safeSiteName)

			err := exporter.ExportToJSON(products, *outputDir, safeSiteName)
			if err != nil {
				log.Errorf("Failed to export products for site %s: %v", siteName, err)
			}
			totalProductsScraped += len(products)
		} else {
			log.Infof("No products to export for site: %s", siteName)
		}
	}
	mapMutex.Unlock()

	log.Infof("EcomCrawler finished. Total products scraped: %d. Output saved to directory: %s", totalProductsScraped, *outputDir)
	log.Infof("Check the '%s' directory for output JSON files.", *outputDir)
}

// Helper function to get absolute path for output (optional)
func getAbsPath(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Warnf("Could not get absolute path for %s: %v", path, err)
		return path
	}
	return absPath
}
