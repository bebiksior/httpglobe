package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bebiksior/httpglobe/pkg/checker"
	"github.com/bebiksior/httpglobe/pkg/config"
	"github.com/bebiksior/httpglobe/pkg/output"
	"github.com/bebiksior/httpglobe/pkg/proxy"
	"github.com/schollz/progressbar/v3"
	"github.com/sirupsen/logrus"
)

type Result struct {
	URL       string              `json:"url"`
	Responses []*checker.Response `json:"responses"`
	HasDiff   bool                `json:"has_differences"`
}

type Output struct {
	Results []*Result `json:"results"`
}

// saveResults saves results to a file
func saveResults(results *Output, outputFile string) error {
	resultJSON, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outputFile, resultJSON, 0644)
}

// checkURL performs concurrent HTTP requests to a URL from different countries
// and returns a Result containing all responses and whether differences were found
func checkURL(url string, proxy *proxy.Proxy, countries []string) *Result {
	var wg sync.WaitGroup
	responses := make([]*checker.Response, len(countries))

	for i, country := range countries {
		wg.Add(1)
		go func(index int, country string) {
			defer wg.Done()
			responses[index] = checker.Check(url, proxy, country)
		}(i, country)

		time.Sleep(50 * time.Millisecond)
	}

	wg.Wait()

	result := &Result{
		URL:       url,
		Responses: responses,
	}
	result.HasDiff = checker.CompareResponses(responses)

	return result
}

func main() {
	configPath := flag.String("config", "", "Path to the configuration file (default: $HOME/.config/httpglobe/config.json)")
	outputFile := flag.String("output", "", "Path to the output JSON file")
	inputFile := flag.String("input", "", "Path to file with URLs (one per line). If not provided, reads from stdin")
	verify := flag.Bool("verify", true, "Verify results by sending requests twice to filter out false positives")
	concurrency := flag.Int("c", 5, "Number of concurrent URL checks")
	flag.Parse()

	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
	})

	cfg, err := config.Load(*configPath, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to load configuration")
	}

	p := proxy.New(
		cfg.Proxy.Host,
		cfg.Proxy.Port,
		cfg.Proxy.Username,
		cfg.Proxy.Password,
	)

	var reader io.Reader
	var inputFileHandle *os.File
	if *inputFile != "" {
		file, err := os.Open(*inputFile)
		if err != nil {
			log.WithError(err).Fatal("Failed to open input file")
		}
		defer file.Close()
		reader = file
		inputFileHandle = file
	} else {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			log.Fatal("No input file provided and no data on stdin")
		}
		reader = os.Stdin
	}

	var totalURLs int
	var urls []string
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		if url != "" {
			totalURLs++
			if inputFileHandle == nil {
				urls = append(urls, url)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.WithError(err).Fatal("Error counting URLs")
	}

	if totalURLs == 0 {
		log.Fatal("No valid URLs found in input")
	}

	if inputFileHandle != nil {
		if _, err := inputFileHandle.Seek(0, 0); err != nil {
			log.WithError(err).Fatal("Error resetting file pointer")
		}
	}

	results := &Output{
		Results: make([]*Result, 0),
	}

	jobs := make(chan string)
	resultsChan := make(chan *Result)
	var wg sync.WaitGroup

	bar := progressbar.NewOptions(totalURLs,
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(30),
		progressbar.OptionSetDescription("[cyan]Processing URLs[reset]"),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]━[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionShowBytes(false),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWriter(os.Stderr),
	)

	bar.Set(0)

	// Start worker goroutines
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range jobs {
				result := checkURL(url, p, cfg.Countries)

				if !result.HasDiff {
					bar.Add(1)
					continue
				}

				if *verify {
					verifyResult := checkURL(url, p, cfg.Countries)

					if !verifyResult.HasDiff {
						resultsChan <- nil
						bar.Add(1)
						continue
					}

					if !checker.ResponsePatternsMatch(result.Responses, verifyResult.Responses) {
						resultsChan <- nil
						bar.Add(1)
						continue
					}
				}

				resultsChan <- result
				bar.Add(1)
			}
		}()
	}

	var resultWg sync.WaitGroup
	resultWg.Add(1)
	go func() {
		defer resultWg.Done()
		isFirstResult := true
		for result := range resultsChan {
			if result == nil {
				continue
			}

			results.Results = append(results.Results, result)
			if !isFirstResult {
				fmt.Println()
			}
			output.PrintResponses(result.URL, result.Responses)
			isFirstResult = false

			if *outputFile != "" {
				if err := saveResults(results, *outputFile); err != nil {
					log.WithError(err).Error("Failed to save results")
				}
			}
		}
	}()

	// Send URLs to jobs channel
	if inputFileHandle != nil {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			url := strings.TrimSpace(scanner.Text())
			if url == "" {
				continue
			}

			if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
				url = "https://" + url
			}

			jobs <- url
		}
	} else {
		// Reading from stored stdin URLs
		for _, url := range urls {
			if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
				url = "https://" + url
			}
			jobs <- url
		}
	}

	if err := scanner.Err(); err != nil {
		log.WithError(err).Fatal("Error reading input")
	}

	close(jobs)
	wg.Wait()

	close(resultsChan)
	resultWg.Wait()

	if *outputFile != "" {
		if err := saveResults(results, *outputFile); err != nil {
			log.WithError(err).Fatal("Failed to save final results")
		}
	}

	_ = bar.Finish()
	log.Info("Processing completed successfully")
}
