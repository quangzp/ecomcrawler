package scraper

import (
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus" // Using logrus for structured logging
)

// cleanString removes extra spaces and newlines from a string.
func cleanString(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	// Replace multiple spaces with a single space
	re := regexp.MustCompile(`\s+`)
	s = re.ReplaceAllString(s, " ")
	return s
}

// extractPrice attempts to extract a numerical value from a price string.
// This is a basic example and might need significant improvement based on actual price formats.
func extractPrice(priceStr string) string {
	priceStr = cleanString(priceStr)
	// Regex to find numbers (integers or decimals)
	// This regex tries to capture common price formats like "1,234.56", "1234.56", "1.234,56" (some European), "1234"
	re := regexp.MustCompile(`[\d\.,]+`)
	match := re.FindString(priceStr)
	if match == "" {
		log.WithField("original_price", priceStr).Warn("Could not extract numerical value from price string")
		return priceStr // Return original if no number found, or handle as error
	}

	// Further cleaning: remove thousands separators (commas or dots depending on locale)
	// Assuming dot as decimal separator for now. If comma is decimal, logic needs adjustment.
	// This is a common source of errors, robust price parsing is hard.
	cleanedMatch := strings.ReplaceAll(match, ",", "") // Remove commas (often thousand separators)

	// If the original match contained a comma and then a dot, it was likely a Euro-style price like 1.234,56
	// If it contained a dot and then a comma, it's unusual, but we'll try to handle it.
	// For simplicity, we'll assume if a comma exists, it's a thousands separator unless it's the *only* non-digit.

	// Try to convert to float to validate, then format back to string if needed, or keep as string.
	// For this example, we'll just return the cleaned numeric string.
	_, err := strconv.ParseFloat(cleanedMatch, 64)
	if err != nil {
		// If parsing fails, it might be due to multiple decimal points or other issues.
		// A more robust solution would try to identify the correct decimal separator.
		log.WithFields(log.Fields{
			"original_price":  priceStr,
			"extracted_match": match,
			"cleaned_match":   cleanedMatch,
			"error":           err,
		}).Warn("Failed to parse cleaned price string to float, returning cleaned match.")
		return match // Return the originally matched numeric part if further cleaning fails
	}

	return cleanedMatch // Return the cleaned numeric string part
}

// extractCategory can be more complex, e.g., taking the last part of a breadcrumb.
// For now, it's a simple clean.
func extractCategory(categoryStr string) string {
	return cleanString(categoryStr)
}
