package output

import (
	"fmt"
	"strings"

	"github.com/bebiksior/httpglobe/pkg/checker"
	"github.com/fatih/color"
)

var (
	green   = color.New(color.FgGreen).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	blue    = color.New(color.FgBlue).SprintFunc()
	magenta = color.New(color.FgMagenta).SprintFunc()
)

func PrintResponses(url string, responses []*checker.Response) {
	fmt.Printf("%s\n", blue(url))

	groups := make(map[string][]*checker.Response)
	for _, resp := range responses {
		key := formatResponseKey(resp)
		groups[key] = append(groups[key], resp)
	}

	// Print each group
	for _, group := range groups {
		countries := make([]string, len(group))
		for i, resp := range group {
			countries[i] = resp.Country
		}

		var statusStr string
		if group[0].Error != "" {
			statusStr = red(group[0].Error)
		} else {
			statusStr = colorizeStatus(group[0].StatusCode)
		}

		fmt.Printf("[%s] [%s] %s\n",
			strings.Join(countries, ","),
			statusStr,
			formatTitle(group[0].Title),
		)
	}
}

// formatResponseKey creates a unique key for grouping similar responses
func formatResponseKey(resp *checker.Response) string {
	if resp.Error != "" {
		return fmt.Sprintf("error:%s", resp.Error)
	}
	return fmt.Sprintf("%d:%s", resp.StatusCode, resp.Title)
}

func colorizeStatus(status int) string {
	switch {
	case status >= 200 && status < 300:
		return green(status)
	case status >= 300 && status < 400:
		return yellow(status)
	case status >= 400 && status < 500:
		return red(status)
	case status >= 500:
		return magenta(status)
	default:
		return fmt.Sprintf("%d", status)
	}
}

func formatTitle(title string) string {
	if title == "" {
		return yellow("no title")
	}
	if len(title) > 50 {
		return title[:47] + "..."
	}
	return title
}
