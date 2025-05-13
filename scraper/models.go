package scraper

import "time"

// Product holds the scraped product data.
type Product struct {
	Name      string    `json:"name"`
	Price     string    `json:"price"` // Using string for price to handle various currency formats initially
	Category  string    `json:"category,omitempty"`
	SourceURL string    `json:"source_url"`
	ScrapedAt time.Time `json:"scraped_at"`
}

// SiteConfig defines the configuration for a single e-commerce site to be scraped.
type SiteConfig struct {
	Name             string        `json:"name"`                         // Informative name for the site
	BaseURL          string        `json:"base_url"`                     // Starting URL for crawling
	ProductSelector  string        `json:"product_selector"`             // CSS selector for individual product items/blocks
	NameSelector     string        `json:"name_selector"`                // CSS selector for product name (relative to ProductSelector)
	PriceSelector    string        `json:"price_selector"`               // CSS selector for product price (relative to ProductSelector)
	CategorySelector string        `json:"category_selector,omitempty"`  // CSS selector for product category (relative to ProductSelector, optional)
	NextPageSelector string        `json:"next_page_selector,omitempty"` // CSS selector for the "next page" link
	MaxDepth         int           `json:"max_depth,omitempty"`          // Max pagination depth (0 for unlimited or if no pagination)
	AllowedDomains   []string      `json:"allowed_domains"`              // Domains the crawler is allowed to visit
	Delay            time.Duration `json:"delay_ms,omitempty"`           // Delay between requests to this domain in milliseconds
	RandomDelay      time.Duration `json:"random_delay_ms,omitempty"`    // Additional random delay
	UserAgent        string        `json:"user_agent,omitempty"`         // Custom User-Agent
	MaxRetries       int           `json:"max_retries,omitempty"`        // Max retries on failed requests
	// Concurrency related settings for Colly
	Async             bool `json:"async,omitempty"`               // Enable asynchronous requests
	Parallelism       int  `json:"parallelism,omitempty"`         // Number of parallel threads per domain
	RobotsTxtDisabled bool `json:"robots_txt_disabled,omitempty"` // Set to true to disable robots.txt
}
