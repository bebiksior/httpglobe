package checker

import (
	"fmt"
	"strings"

	"github.com/Danny-Dasilva/CycleTLS/cycletls"
	"github.com/bebiksior/httpglobe/pkg/proxy"
)

// Response represents an HTTP response with relevant information
type Response struct {
	StatusCode    int    `json:"status_code"`
	Title         string `json:"title"`
	ContentLength int    `json:"content_length"`
	Country       string `json:"country"`
	Error         string `json:"error,omitempty"`
}

const (
	CHROME_JA3_SIGNATURE = "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-21,29-23-24,0"
)

// Check performs an HTTP request to the given URL using the proxy and country code.
func Check(url string, proxy *proxy.Proxy, country string) *Response {
	resp := &Response{Country: country}

	proxyURL, err := proxy.URL(country)
	if err != nil {
		resp.Error = fmt.Sprintf("creating proxy URL: %v", err)
		return resp
	}

	client := cycletls.Init()
	options := cycletls.Options{
		Ja3:       CHROME_JA3_SIGNATURE,
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Proxy:     *proxyURL,
		Headers: map[string]string{
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
			"Accept-Language": "en-US,en;q=0.5",
			"Accept-Encoding": "gzip, deflate, br",
		},
		Timeout: 30,
	}

	response, err := client.Do(url, options, "GET")
	if err != nil {
		resp.Error = fmt.Sprintf("making request: %v", err)
		return resp
	}

	resp.ContentLength = len(response.Body)
	resp.StatusCode = response.Status
	if title, err := extractTitle(response.Body); err == nil {
		resp.Title = title
	}

	return resp
}

// extractTitle extracts the title from HTML content
func extractTitle(body string) (string, error) {
	titleStart := strings.Index(body, "<title>")
	if titleStart == -1 {
		return "", fmt.Errorf("no title tag found")
	}
	titleStart += 7

	titleEnd := strings.Index(body[titleStart:], "</title>")
	if titleEnd == -1 {
		return "", fmt.Errorf("no closing title tag found")
	}

	return strings.TrimSpace(body[titleStart : titleStart+titleEnd]), nil
}

// CompareResponses compares responses from different countries and returns true
// if there are significant differences in status codes, titles, content length
// (ignoring small differences up to 20%), or error strings.
func CompareResponses(responses []*Response) bool {
	if len(responses) <= 1 {
		return false
	}

	// Use the first response without an error as the reference.
	var reference *Response
	for _, resp := range responses {
		if resp.Error == "" {
			reference = resp
			break
		}
	}

	if reference == nil {
		reference = responses[0]
	}

	// Helper function to determine if the content length difference is significant
	isContentLengthSignificant := func(length1, length2 int) bool {
		if length1 == 0 {
			return length2 != 0
		}

		diff := 100 * abs(length1-length2) / length1
		return diff > 20 // only flag differences >20%
	}

	for _, resp := range responses {
		if resp == reference {
			continue
		}

		if resp.Error != reference.Error {
			return true
		}

		if resp.StatusCode != reference.StatusCode {
			return true
		}

		if resp.Title != reference.Title {
			return true
		}

		if (resp.ContentLength == 0) != (reference.ContentLength == 0) ||
			(resp.ContentLength != 0 && isContentLengthSignificant(reference.ContentLength, resp.ContentLength)) {
			return true
		}
	}

	return false
}

// ResponsePatternsMatch compares two sets of responses to check if they show
// the same pattern of differences across countries. It includes the error,
// status code, title, and content length (if the difference is significant).
func ResponsePatternsMatch(responses1, responses2 []*Response) bool {
	if len(responses1) != len(responses2) {
		return false
	}

	pattern1 := make(map[string]string)
	pattern2 := make(map[string]string)

	for _, resp := range responses1 {
		pattern1[resp.Country] = buildPattern(resp)
	}

	for _, resp := range responses2 {
		pattern2[resp.Country] = buildPattern(resp)
	}

	for country, pat := range pattern1 {
		if pattern2[country] != pat {
			return false
		}
	}

	return true
}

// buildPattern builds a string representation for a response,
func buildPattern(resp *Response) string {
	if resp.Error != "" {
		return fmt.Sprintf("error:%s", resp.Error)
	}

	return fmt.Sprintf("%d:%s:%d", resp.StatusCode, resp.Title, resp.ContentLength)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
