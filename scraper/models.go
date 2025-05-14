package scraper

import "time"

type Product struct {
	Name      string    `json:"name"`
	Price     string    `json:"price"`
	Category  string    `json:"category,omitempty"`
	SourceURL string    `json:"source_url"`
	ScrapedAt time.Time `json:"scraped_at"`
}

type SiteConfig struct {
	Name                     string        `json:"name"`
	BaseURL                  string        `json:"base_url"`
	ProductSelector          string        `json:"product_selector"`
	NameSelector             string        `json:"name_selector"`
	PriceSelector            string        `json:"price_selector"`
	CategorySelector         string        `json:"category_selector,omitempty"`
	NextPageSelector         string        `json:"next_page_selector,omitempty"`
	MaxDepth                 int           `json:"max_depth,omitempty"`
	AllowedDomains           []string      `json:"allowed_domains"`
	Delay                    time.Duration `json:"delay_ms,omitempty"`
	RandomDelay              time.Duration `json:"random_delay_ms,omitempty"`
	UserAgent                string        `json:"user_agent,omitempty"`
	MaxRetries               int           `json:"max_retries,omitempty"`
	Async                    bool          `json:"async,omitempty"`
	Parallelism              int           `json:"parallelism,omitempty"`
	RobotsTxtDisabled        bool          `json:"robots_txt_disabled,omitempty"`
	LoadMoreButtonSelector   string        `json:"load_more_button_selector,omitempty"`
	MaxLoadMoreClicks        int           `json:"max_load_more_clicks,omitempty"`
	ScrollToBottom           bool          `json:"scroll_to_bottom,omitempty"`
	WaitAfterLoadMoreMs      int           `json:"wait_after_load_more_ms,omitempty"`
	ProductContainerSelector string        `json:"product_container_selector,omitempty"`
	Headless                 bool          `json:"headless,omitempty"`
	ChromedpTimeoutSec       int           `json:"chromedp_timeout_sec,omitempty"`
	PollForProductIncrease   bool          `json:"poll_for_product_increase,omitempty"`
	PollTimeoutMs            int           `json:"poll_timeout_ms,omitempty"`
	PollIntervalMs           int           `json:"poll_interval_ms,omitempty"`
}
