package config

import (
	"ecomcrawler/scraper"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

func LoadSiteConfigs(filePath string) ([]scraper.SiteConfig, error) {
	// Read the file content
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}

	// Unmarshal JSON data into a slice of SiteConfig
	var configs []scraper.SiteConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config JSON from %s: %w", filePath, err)
	}

	// Apply default values or perform validation if necessary
	for i := range configs {
		if configs[i].UserAgent == "" {
			configs[i].UserAgent = "EcomCrawler/1.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)" //
		}
		if configs[i].Delay == 0 {
			configs[i].Delay = 1000 * time.Millisecond // Default 1 second delay
		}
		if configs[i].Parallelism == 0 {
			configs[i].Parallelism = 2 // Default parallelism
		}
		if configs[i].MaxDepth == 0 && configs[i].NextPageSelector != "" {
			configs[i].MaxDepth = 5 // Default max depth for pagination if not set
		}
	}

	return configs, nil
}
