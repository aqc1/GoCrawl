package main

/*
TODO:
- Optimize the Following:
    - Checking if URLs were found
    - Scraping each URL
*/

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Web Crawler Struct
type WebCrawler struct {
	mut     sync.Mutex
	visited []string
}

// Struct to Make Checking if -output was Set
type fileFlag struct {
	set  bool
	file string
}

func (ff *fileFlag) Set(val string) error {
	ff.file = val
	ff.set = true
	return nil
}

func (ff *fileFlag) String() string {
	return ff.file
}

func main() {
	// Set Up Waitgroup
	var wg sync.WaitGroup

	// CLI Arg
	var urlToCrawl string
	var sortList bool
	var outputFile fileFlag

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

	flag.Var(
		&outputFile,
		"output",
		"File to Output to",
	)

	flag.Parse()

	// Create crawler
	crawler := WebCrawler{
		visited: []string{},
	}

	// Crawl first page
	page, err := getPage(urlToCrawl)
	if err != nil {
		log.Println(err)
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
	// Sorting URLs
	if sortList {
		sort.Strings(crawler.visited)
	}

	// Creating Output File
	if outputFile.set {
		f, err := os.Create(outputFile.file)
		if err != nil {
			log.Println(err)
		}
		defer f.Close()
		for _, item := range crawler.visited {
			line := fmt.Sprintf("%s\n", item)
			if _, err := f.WriteString(line); err != nil {
				log.Println(err)
			}
		}

	} else {
		// Output URLs
		for _, item := range crawler.visited {
			fmt.Println(item)
		}
	}
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
