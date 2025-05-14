package scraper

import (
	"regexp"
	"strconv"
	"strings"
)

func cleanString(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	re := regexp.MustCompile(`\s+`)
	s = re.ReplaceAllString(s, " ")
	return s
}

func extractPrice(priceStr string) string {
	priceStr = cleanString(priceStr)

	re := regexp.MustCompile(`[\d\.,]+`)
	match := re.FindString(priceStr)
	if match == "" {
		// log.WithField("original_price", priceStr).Warn("Could not extract numerical value from price string")
		return priceStr
	}

	cleanedMatch := strings.ReplaceAll(match, ",", "")
	cleanedMatch = strings.ReplaceAll(match, ".", "")

	_, err := strconv.ParseFloat(cleanedMatch, 64)
	if err != nil {
		// log.WithFields(log.Fields{
		// 	"original_price":  priceStr,
		// 	"extracted_match": match,
		// 	"cleaned_match":   cleanedMatch,
		// 	"error":           err,
		// }).Warn("Failed to parse cleaned price string to float, returning cleaned match.")
		return match
	}

	return cleanedMatch
}

func extractCategory(categoryStr string) string {
	return cleanString(categoryStr)
}
