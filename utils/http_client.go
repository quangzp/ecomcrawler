package utils

import (
	"net/http"
	"time"
)

// DefaultHTTPClient creates a new http.Client with some default settings.
// This can be expanded to include custom transport for retries, throttling, etc.
func DefaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second, // Set a reasonable timeout
		// Transport: &http.Transport{
		// 	// Configure proxy, TLS settings, etc., if needed
		// 	MaxIdleConns:        100,
		// 	IdleConnTimeout:     90 * time.Second,
		// 	TLSHandshakeTimeout: 10 * time.Second,
		// },
	}
}

// GetWithCustomClient performs an HTTP GET request using a provided client.
// You might add more helper functions here for POST requests or requests with custom headers.
// func GetWithCustomClient(client *http.Client, url string) (*http.Response, error) {
// 	req, err := http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		return nil, err
// 	}
// 	// Set common headers, e.g., User-Agent
// 	req.Header.Set("User-Agent", "EcomCrawler/1.0 (+http://your-crawler-info-page.com)")
// 	// Add more headers if necessary
//
// 	return client.Do(req)
// }

// ThrottledTransport can be a custom http.RoundTripper that implements throttling.
// For example:
//
// import "golang.org/x/time/rate"
//
// type ThrottledTransport struct {
//     Transport http.RoundTripper
//     Limiter   *rate.Limiter
// }
//
// func (t *ThrottledTransport) RoundTrip(req *http.Request) (*http.Response, error) {
//     err := t.Limiter.Wait(req.Context()) // Wait for the rate limiter
//     if err != nil {
//         return nil, err
//     }
//     return t.Transport.RoundTrip(req)
// }
//
// func NewThrottledClient(requestsPerSecond float64, burst int) *http.Client {
// 	limiter := rate.NewLimiter(rate.Limit(requestsPerSecond), burst)
// 	return &http.Client{
// 		Transport: &ThrottledTransport{
// 			Transport: http.DefaultTransport, // Or your custom base transport
// 			Limiter:   limiter,
// 		},
// 		Timeout: 30 * time.Second,
// 	}
// }
//
// Example usage in scraper:
// s.httpClient = utils.NewThrottledClient(1, 1) // 1 request per second
