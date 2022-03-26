package main

/*
TODO:
    - Optimize:
        - Checking if URLs were found
        - Scraping each URL
    - Add:
        - Sorting Option for Found URLs
        - Output Option to Send to File (Sort first if desired)
*/

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
)

type WebCrawler struct {
	mut     sync.Mutex
	visited []string
}

func main() {
	// Set Up Waitgroup
	var wg sync.WaitGroup

	// CLI Arg
	var urlToCrawl string
	var sortList bool

	flag.StringVar(
		&urlToCrawl,
		"url",
		"https://127.0.0.1/",
		"URL to Start Crawling At",
	)

	flag.BoolVar(
		&sortList,
		"sort",
		false,
		"Sort URLs",
	)

	flag.Parse()

	// Create crawler
	crawler := WebCrawler{
		visited: []string{},
	}

	// Crawl first page
	page, err := getPage(urlToCrawl)
	if err != nil {
		usage()
	}
	scrapePage(page, &crawler, nil)

	// Concurrently Scrape New Pages
	for {
		// There's probably a more optimized to do this...but it works
		// TODO: Optimize this
		tmp := make([]string, len(crawler.visited))
		copy(tmp, crawler.visited)

		// For every found URL, spin up a go func
		// TODO: Optimize this
		for _, url := range crawler.visited {
			wg.Add(1)
			go func(currentUrl string) {
				insidePage, err := getPage(currentUrl)
				if err == nil {
					scrapePage(insidePage, &crawler, &wg)
				} else {
					wg.Done()
				}
			}(url)
		}
		wg.Wait()

		// Check if no new URLs are found
		// Find a better way to verify this (?)
		if checkEqual(tmp, crawler.visited) {
			break
		}
	}

	// Output Items
	// Probably do this: go run main.go -url URL | sort | tee found_urls.txt
	if sortList {
		sort.Strings(crawler.visited)
	}
	for _, item := range crawler.visited {
		fmt.Println(item)
	}
}

// Explains Usage when Something Goes Wrong (Probably Forgot a Flag)
func usage() {
	fmt.Println("[+] Usage: ./GoCrawl -url URL")
	fmt.Println("[!] Default URL: https://127.0.0.1/")
	os.Exit(1)
}

// Returns Byte Array of Page Source
func getPage(url string) ([]byte, error) {
	// Grab page
	resp, err := http.Get(url)
	if err != nil {
		return []byte{}, err
	}

	// Grab body of page
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return []byte{}, err
	}

	return body, nil
}

// Extracts URLs from Page Body
func scrapePage(page []byte, crawler *WebCrawler, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}

	// Search for http or https
	re := regexp.MustCompile(`<a href="(http|https)(.*?)>`)
	match := re.FindAllStringSubmatch(string(page), -1)

	// Extract just the URL
	crawler.mut.Lock()
	for _, element := range match {
		url := strings.Replace(strings.Replace(element[0], "<a href=", "", -1), ">", "", -1)
		trimmed := trimFromSpace(url)
		if !checkIfVisited(trimmed, crawler) {
			crawler.visited = append(crawler.visited, trimmed)
		}
	}
	crawler.mut.Unlock()
}

// See if URL was Already Found
func checkIfVisited(testString string, crawler *WebCrawler) bool {
	for _, url := range crawler.visited {
		if url == testString {
			return true
		}
	}
	return false
}

// Compare Two Slices
func checkEqual(tmp, crawled []string) bool {
	if len(tmp) != len(crawled) {
		return false
	}
	for indx, val := range tmp {
		if val != crawled[indx] {
			return false
		}
	}
	return true
}

// Trim Off Extra Stuff...
func trimFromSpace(toTrim string) string {
	if indx := strings.Index(toTrim, " "); indx != -1 {
		return toTrim[:indx]
	}
	return toTrim
}
