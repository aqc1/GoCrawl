package main

import (
    "flag"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "regexp"
    "strings"
    "sync"
)

type WebCrawler struct {
    mut         sync.Mutex
    visited     []string
}

func main() {
    // Set Up Waitgroup/Mutex
    var wg sync.WaitGroup

    // CLI Arg
    var urlToCrawl string
    flag.StringVar(&urlToCrawl, "url", "https://127.0.0.1/", "URL to Start Crawling At")
    flag.Parse()

    // Initial Crawl
    crawler := WebCrawler{
        visited: []string{},
    }
    page, err := getPage(urlToCrawl)
    if err != nil {
        usage()
    }
    scrapePage(page, &crawler, nil)

    // Concurrently Scrape New Pages
    for {
        tmp := make([]string, len(crawler.visited))
        copy(tmp, crawler.visited)
        for _, url := range crawler.visited {
            go func(currentUrl string) {
                wg.Add(1)
                insidePage, err := getPage(currentUrl)
                if err == nil {
                    scrapePage(insidePage, &crawler, &wg)
                } else {
                    wg.Done()
                }
            }(url)
        }
        wg.Wait()
        if checkEqual(tmp, crawler.visited) {
            break
        }
    }

    // Output Items
    // Probably do this: go run main.go -url URL | sort > output.txt
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
func getPage(url string, ) ([]byte, error) {
    resp, err := http.Get(url)
    if err != nil {
        return []byte{}, err
    }

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
    re := regexp.MustCompile(`<a href="(http|https)(.*?)>`)
    match := re.FindAllStringSubmatch(string(page), -1)
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
