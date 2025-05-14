package exporter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ecomcrawler/scraper"

	log "github.com/sirupsen/logrus"
)

// ExportToJSON saves the list of products to a JSON file.
// The filename will include a timestamp.
func ExportToJSON(products []scraper.Product, outputDir string, siteName string) error {
	if len(products) == 0 {
		log.WithField("site", siteName).Info("No products found to export.")
		return nil
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	fileName := fmt.Sprintf("%s_%s_products.json", siteName, timestamp)
	filePath := filepath.Join(outputDir, fileName)

	jsonData, err := json.MarshalIndent(products, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal products to JSON for site %s: %w", siteName, err)
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON to file %s for site %s: %w", filePath, siteName, err)
	}

	log.WithFields(log.Fields{
		"site":  siteName,
		"file":  filePath,
		"count": len(products),
	}).Info("Successfully exported products to JSON.")
	return nil
}
