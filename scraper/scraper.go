package scraper

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	log "github.com/sirupsen/logrus"
)

func CrawlSite(siteCfg SiteConfig, productChan chan<- Product, wg *sync.WaitGroup) {
	defer wg.Done()

	logger := log.WithFields(log.Fields{
		"site": siteCfg.Name,
		"url":  siteCfg.BaseURL,
	})

	logger.Info("Starting crawl")
	crawlSiteWithChromedp(siteCfg, productChan, logger)
}

func crawlSiteWithChromedp(siteCfg SiteConfig, productChan chan<- Product, logger *log.Entry) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		// chromedp.WindowSize(600, 400),
		chromedp.UserAgent(siteCfg.UserAgent),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	timeout := time.Duration(siteCfg.ChromedpTimeoutSec) * time.Second
	if siteCfg.ChromedpTimeoutSec == 0 {
		timeout = 60 * time.Second
	}
	taskCtx, cancelTask := context.WithTimeout(allocCtx, timeout)
	defer cancelTask()

	browserCtx, cancelBrowser := chromedp.NewContext(taskCtx, chromedp.WithLogf(logger.Printf))
	defer cancelBrowser()

	var siteProducts []Product
	var mu sync.Mutex

	initialActions := []chromedp.Action{
		emulation.SetDeviceMetricsOverride(int64(1280), int64(800), 1, false),
	}
	initialActions = append(initialActions, chromedp.Navigate(siteCfg.BaseURL))
	initialActions = append(initialActions, chromedp.WaitVisible(siteCfg.ProductContainerSelector, chromedp.ByQuery))

	logger.Infof("Navigating to %s", siteCfg.BaseURL)
	if err := chromedp.Run(browserCtx,
		initialActions...,
	); err != nil {
		logger.WithError(err).Error("Failed to navigate to base URL or find initial product container")
		return
	}
	logger.Info("Initial page loaded.")

	for i := 0; i < siteCfg.MaxLoadMoreClicks || siteCfg.MaxLoadMoreClicks == 0; i++ {
		var htmlContent string
		err := chromedp.Run(browserCtx,
			chromedp.OuterHTML(siteCfg.ProductContainerSelector, &htmlContent, chromedp.ByQuery),
		)
		if err != nil {
			logger.WithError(err).Warn("Failed to get HTML content from product container")
		} else {

			//doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
			// content := []byte(htmlContent)
			// outp := "output" + strconv.Itoa(i) + ".html"
			// os.WriteFile(outp, content, 0644)

			parseAndAddProductsFromHTML(htmlContent, siteCfg, &siteProducts, &mu, logger, siteCfg.BaseURL)
		}

		if siteCfg.LoadMoreButtonSelector == "" {
			logger.Info("No 'Load More' button selector configured. Stopping.")
			break
		}

		var buttonNodes []*cdp.Node
		err = chromedp.Run(browserCtx,
			chromedp.Nodes(siteCfg.LoadMoreButtonSelector, &buttonNodes, chromedp.ByQuery),
		)
		if err != nil || len(buttonNodes) == 0 {
			logger.Info("Load More button not found or error acquiring it. Assuming no more products.")
			break
		}

		time.Sleep(siteCfg.Delay)
		if siteCfg.ScrollToBottom {
			logger.Debug("Scrolling to bottom")
			if err := chromedp.Run(browserCtx, dom.ScrollIntoViewIfNeeded()); err != nil {
				logger.WithError(err).Warn("Failed to scroll to bottom")
			}
		} else {
			logger.Debugf("Scrolling button '%s' into view", siteCfg.LoadMoreButtonSelector)
			if err := chromedp.Run(browserCtx, chromedp.ScrollIntoView(siteCfg.LoadMoreButtonSelector, chromedp.ByQuery)); err != nil {
				logger.WithError(err).Warn("Failed to scroll 'Load More' button into view")
			}
		}

		time.Sleep(1000 * time.Millisecond)

		logger.Infof("Attempting to click 'Load More' button (attempt %d/%d)", i+1, siteCfg.MaxLoadMoreClicks)
		err = chromedp.Run(browserCtx,
			chromedp.Click(siteCfg.LoadMoreButtonSelector, chromedp.ByQuery, chromedp.NodeVisible),
		)
		if err != nil {
			logger.WithError(err).Warn("Failed to click 'Load More' button. Assuming no more products or button is gone.")
			break
		}

		waitDuration := time.Duration(siteCfg.WaitAfterLoadMoreMs) * time.Millisecond
		if siteCfg.WaitAfterLoadMoreMs == 0 {
			waitDuration = 3 * time.Second
		}
		logger.Infof("Clicked 'Load More'. Waiting for %v for new content...", waitDuration)

		chromedp.Sleep(waitDuration)

		if siteCfg.MaxLoadMoreClicks > 0 && i+1 == siteCfg.MaxLoadMoreClicks {
			logger.Info("Reached max 'Load More' clicks.")
			var finalHTMLContent string
			finalErr := chromedp.Run(browserCtx,
				chromedp.OuterHTML(siteCfg.ProductContainerSelector, &finalHTMLContent, chromedp.ByQuery),
			)
			if finalErr == nil {
				//doc, parseErr := goquery.NewDocumentFromReader(strings.NewReader(finalHTMLContent))
				parseAndAddProductsFromHTML(finalHTMLContent, siteCfg, &siteProducts, &mu, logger, siteCfg.BaseURL)
			}
			break
		}
	}
	for _, product := range siteProducts {
		// logger.Info("Sent product: ", product.Name, " - ", product.SourceURL)
		productChan <- product
	}

	logger.WithField("products_found_chromedp", len(siteProducts)).Info("Finished crawl for site using Chromedp")
}

func parseAndAddProductsFromHTML(htmlContent string, siteCfg SiteConfig, siteProducts *[]Product, mu *sync.Mutex, logger *log.Entry, pageURL string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		logger.WithError(err).Warn("Failed to parse HTML content with goquery during product extraction")
		return
	}

	foundProductsInThisParse := 0
	doc.Find(siteCfg.ProductSelector).Each(func(idx int, s *goquery.Selection) {
		productName := cleanString(s.Find(siteCfg.NameSelector).First().Text())
		productPrice := extractPrice(s.Find(siteCfg.PriceSelector).First().Text())
		productCategory := ""
		if siteCfg.CategorySelector != "" {
			productCategory = extractCategory(s.Find(siteCfg.CategorySelector).First().Text())
		}

		var productLink string
		nameLinkSelection := s.Find(siteCfg.NameSelector).First()
		if nameLinkSelection.Is("a") {
			productLink, _ = nameLinkSelection.Attr("href")
		} else {
			nameLinkSelection.Closest("a").Each(func(_ int, a *goquery.Selection) {
				productLink, _ = a.Attr("href")
			})
			if productLink == "" {
				s.Find("a").Each(func(_ int, a *goquery.Selection) {
					if tempLink, exists := a.Attr("href"); exists && productLink == "" {
						productLink = tempLink
					}
				})
			}
		}

		if productLink != "" && !strings.HasPrefix(productLink, "http") {
			base, errBase := url.Parse(pageURL)
			rel, errRel := url.Parse(productLink)
			if errBase == nil && errRel == nil && base != nil && rel != nil {
				productLink = base.ResolveReference(rel).String()
			} else {
				logger.Warnf("Could not resolve relative product URL: %s (base: %s)", productLink, pageURL)
			}
		}
		if productLink == "" {
			productLink = pageURL // Fallback to the page URL if no specific product link found
		}

		if productName != "" || productPrice != "" {
			mu.Lock()
			isDuplicate := false
			for _, p := range *siteProducts {
				if p.Name == productName && p.Price == productPrice && p.SourceURL == productLink {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				newProduct := Product{
					Name:      productName,
					Price:     productPrice,
					Category:  productCategory,
					SourceURL: productLink,
					ScrapedAt: time.Now().UTC(),
				}
				*siteProducts = append(*siteProducts, newProduct)
				foundProductsInThisParse++
			}
			mu.Unlock()
		}
	})
	if foundProductsInThisParse > 0 {
		logger.Infof("Parsed and added %d new unique products from current HTML state.", foundProductsInThisParse)
	}
}
