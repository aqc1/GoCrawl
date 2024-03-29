package main

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

// WebCrawler Struct - visited: list of visited URLs
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
	var maxDepth int

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

	flag.IntVar(
		&maxDepth,
		"depth",
		1,
		"Max Depth of Crawling",
	)

	flag.Parse()

	// Create crawler
	crawler := WebCrawler{
		visited: []string{},
	}

	// Crawl first page
	timesCrawled := 0
	page, err := getPage(urlToCrawl)
	if err != nil {
		log.Println(err)
	}
	timesCrawled++
	scrapePage(page, &crawler, nil)

	// Concurrently Scrape New Pages
	newlyFound := make([]string, len(crawler.visited))
	copy(newlyFound, crawler.visited)
	for timesCrawled < maxDepth {
		currentlyFound := make([]string, len(crawler.visited))
		copy(currentlyFound, crawler.visited)

		// For every found URL, spin up a thread
		for _, url := range newlyFound {
			wg.Add(1)
			go func(currentUrl string) {
				insidePage, err := getPage(currentUrl[1 : len(currentUrl)-1])
				if err == nil {
					scrapePage(insidePage, &crawler, &wg)
				} else {
					wg.Done()
				}
			}(url)
		}
		wg.Wait()
		timesCrawled++

		// Check if no new URLs are found
		// Find which URLs are new...only scrape them
		if checkEqual(currentlyFound, crawler.visited) {
			break
		} else {
			newlyFound = crawler.visited[len(newlyFound):]
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
// :param url: URL to get the response from
// :return: body of the response, errors
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
// :param page: Page to scrape URLs from
// :param crawler: WebCrawler struct
// :param wg: Wait Group for async
func scrapePage(page []byte, crawler *WebCrawler, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	// Search for http or https
	reLink := regexp.MustCompile(`<a( (.*)?=(.*?))* href="(http|https)(.*?)">`)
	reReference := regexp.MustCompile(`<a(.*)?href=`)
	match := reLink.FindAllStringSubmatch(string(page), -1)

	// Extract just the URL
	crawler.mut.Lock()
	for _, element := range match {
		url := strings.Replace(reReference.ReplaceAllString(element[0], ""), ">", "", -1)
		trimmed := trimFromSpace(url)
		if !checkIfVisited(trimmed, crawler) {
			crawler.visited = append(crawler.visited, trimmed)
		}
	}
	crawler.mut.Unlock()
}

// See if URL was Already Found
// :param testString: string to test if it was already found
// :param crawler: WebCrawler struct
// :return: bool - was the URL visited already?
func checkIfVisited(testString string, crawler *WebCrawler) bool {
	for _, url := range crawler.visited {
		if url == testString {
			return true
		}
	}
	return false
}

// Compare Two Slices
// :param tmp: temporary list of visited URLs
// :param crawled: array of already visited URLs
// :return: bool - are these arrays equivalent?
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
// :param toTrim: string to trim down
// :return: trimmed string
func trimFromSpace(toTrim string) string {
	if indx := strings.Index(toTrim, " "); indx != -1 {
		return toTrim[:indx]
	}
	return toTrim
}
