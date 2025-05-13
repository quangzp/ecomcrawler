package scraper

import (
	"net/url"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/queue"
	log "github.com/sirupsen/logrus"
)

// CrawlSite performs the web scraping for a single configured site.
// It sends found products to the productChan and signals completion via wg.
func CrawlSite(siteCfg SiteConfig, productChan chan<- Product, wg *sync.WaitGroup) {
	defer wg.Done() // Signal that this goroutine is done when the function returns

	logger := log.WithFields(log.Fields{
		"site": siteCfg.Name,
		"url":  siteCfg.BaseURL,
	})
	logger.Info("Starting crawl")

	// Instantiate default collector
	c := colly.NewCollector(
		// colly.UserAgent(siteCfg.UserAgent), // Set globally or per request
		colly.AllowedDomains(siteCfg.AllowedDomains...),
		colly.MaxDepth(siteCfg.MaxDepth), // MaxDepth defines the limit of recursion for links
		colly.Async(siteCfg.Async),       // Enable asynchronous requests for better performance
	)

	// Configure retries
	// if siteCfg.MaxRetries > 0 {
	// 	c.MaxRetries = siteCfg.MaxRetries
	// }

	// Configure rate limiting and parallelism
	limitRule := &colly.LimitRule{
		DomainGlob:  "*", // Apply to all domains specified in AllowedDomains
		Parallelism: siteCfg.Parallelism,
		Delay:       siteCfg.Delay,
		RandomDelay: siteCfg.RandomDelay,
	}
	if err := c.Limit(limitRule); err != nil {
		logger.WithError(err).Error("Failed to set limit rule")
		return
	}

	// Disable robots.txt if configured
	if siteCfg.RobotsTxtDisabled {
		c.IgnoreRobotsTxt = true
	}

	// Create a request queue with a specific thread count (if not using Async directly)
	// If using c.Async = true, Colly manages its own concurrency.
	// The queue is more for managing the URLs to visit if you have many starting points or want finer control.
	// For this setup, with a single BaseURL and pagination, Async + Parallelism in LimitRule is often sufficient.
	// However, if you had many initial URLs for a single site config, a queue could be useful.
	q, _ := queue.New(
		siteCfg.Parallelism,                         // Number of consumer threads
		&queue.InMemoryQueueStorage{MaxSize: 10000}, // Max URLs in queue
	)

	// Slice to hold products for this specific site crawl
	var siteProducts []Product

	// Mutex for safely appending to siteProducts if multiple OnHTML callbacks run concurrently for the same collector
	// (though with Colly's default handling for a single collector instance, direct concurrent writes to siteProducts
	// from OnHTML for the *same* collector are less of a concern than if you were manually managing goroutines per element).
	// However, if Async is true, callbacks can execute concurrently.
	var mu sync.Mutex

	// OnHTML callback for finding product items
	c.OnHTML(siteCfg.ProductSelector, func(e *colly.HTMLElement) {
		// Set User-Agent per request if needed, or rely on global
		e.Request.Headers.Set("User-Agent", siteCfg.UserAgent)

		productName := cleanString(e.ChildText(siteCfg.NameSelector))
		productPrice := extractPrice(e.ChildText(siteCfg.PriceSelector)) // Use the parser utility
		productCategory := ""
		if siteCfg.CategorySelector != "" {
			productCategory = extractCategory(e.ChildText(siteCfg.CategorySelector))
		}
		productURL := e.Request.AbsoluteURL(e.ChildAttr("a", "href")) // Assuming name is in an <a> tag for its URL
		if productName == "" && productPrice == "" {                  // Skip if essential data is missing
			logger.WithField("element_html", e.Text).Debug("Skipping element, missing name and price.")
			return
		}

		mu.Lock()
		product := Product{
			Name:      productName,
			Price:     productPrice,
			Category:  productCategory,
			SourceURL: productURL, // Or e.Request.URL.String() for the page URL it was found on
			ScrapedAt: time.Now().UTC(),
		}
		siteProducts = append(siteProducts, product)
		mu.Unlock()

		logger.WithFields(log.Fields{"name": productName, "price": productPrice}).Debug("Found product")
	})

	// OnHTML callback for finding the "next page" link for pagination
	if siteCfg.NextPageSelector != "" {
		c.OnHTML(siteCfg.NextPageSelector, func(e *colly.HTMLElement) {
			nextPageLink := e.Request.AbsoluteURL(e.Attr("href"))
			if nextPageLink != "" {
				logger.WithField("next_page_url", nextPageLink).Info("Found next page")
				// To prevent re-visiting and to respect MaxDepth, Colly handles this if MaxDepth > 0.
				// If MaxDepth is 1, it only crawls the initial page. If > 1, it follows links.
				// We simply visit the link. Colly's MaxDepth will stop it.
				time.Sleep(siteCfg.Delay / 2) // Optional small delay before queuing next page
				err := q.AddURL(nextPageLink) // Add to queue if using queue explicitly
				if err != nil {
					logger.WithError(err).Warn("Failed to add next page to queue")
				}
				// Or directly: e.Request.Visit(nextPageLink) if not using explicit queue for pagination
			}
		})
	}

	// OnRequest callback to set headers or log
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", siteCfg.UserAgent) // Ensure User-Agent is set
		logger.WithField("url", r.URL.String()).Info("Visiting page")
	})

	// OnError callback to handle errors during requests
	c.OnError(func(r *colly.Response, err error) {
		logger.WithFields(log.Fields{
			"url":         r.Request.URL.String(),
			"status_code": r.StatusCode,
			"error":       err,
		}).Error("Request failed")
		// Implement retry logic here if not using Colly's built-in retries, or for specific error types
	})

	// OnScraped callback, after OnHTML has finished
	c.OnScraped(func(r *colly.Response) {
		logger.WithField("url", r.Request.URL.String()).Info("Finished scraping page")
	})

	// Start scraping by adding the base URL to the queue
	err := q.AddURL(siteCfg.BaseURL)
	if err != nil {
		logger.WithError(err).Errorf("Failed to add base URL to queue: %s", siteCfg.BaseURL)
		return
	}

	// Start consumer threads and wait for them to finish (if using explicit queue)
	// If only using c.Visit and c.Async, c.Wait() is the primary mechanism.
	q.Run(c) // This will block until the queue is empty and workers are done.
	c.Wait() // Wait for all asynchronous operations of the collector to complete.

	// Send all collected products for this site to the main channel
	mu.Lock() // Ensure thread-safe access to siteProducts, though by this point, callbacks should be done.
	for _, product := range siteProducts {
		productChan <- product
	}
	mu.Unlock()

	logger.WithField("products_found", len(siteProducts)).Info("Finished crawl for site")
}

// Helper to make relative URLs absolute
func absoluteURL(baseURL, relativePath string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return relativePath // Should not happen with valid BaseURL
	}
	rel, err := url.Parse(relativePath)
	if err != nil {
		return relativePath // Malformed relative path
	}
	return base.ResolveReference(rel).String()
}
