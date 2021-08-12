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
    visited     []string
}

func main() {
    // Set Up Waitgroup/Mutex
    var wg sync.WaitGroup
    var mut sync.Mutex

    // CLI Arg
    // TODO: Have to Come up With Better Default Option...
    var urlToCrawl string
    flag.StringVar(&urlToCrawl, "url", "", "URL to Start Crawling At")
    flag.Parse()

    // Initial Crawl
    crawler := WebCrawler{[]string{}}
    page, err := getPage(urlToCrawl)
    if err != nil {
        usage()
    }
    scrapePage(page, &crawler, &mut, nil)

    // Concurrently Scrape New Pages
    for{
        tmp := make([]string, len(crawler.visited))
        copy(tmp, crawler.visited)
        for _, url := range crawler.visited{
            go func(currentUrl string) {
                wg.Add(1)
                insidePage, err := getPage(currentUrl)
                if err == nil{
                    scrapePage(insidePage, &crawler, &mut, &wg)
                } else {
                    wg.Done()
                }
            }(url)
        }
        wg.Wait()
        if checkEqual(tmp, crawler.visited){
            break
        }
    }

    // Output Items
    // TODO: Do this but with Go --> Probably Pipe to sort...
    // `go run main.go | sort`
    for _, item := range crawler.visited{
        fmt.Println(item)
    }
}

// Explains Usage when Something Goes Wrong (Probably Forgot a Flag)
func usage() {
    fmt.Println("[+] Usage: ./GoCrawl -url URL")
    os.Exit(1)
}

// Returns Byte Array of Page Source
func getPage(url string, ) ([]byte, error){
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
func scrapePage(page []byte, crawler *WebCrawler, mut *sync.Mutex, wg *sync.WaitGroup){
    re := regexp.MustCompile(`<a href="(http|https)(.*?)>`)
    match := re.FindAllStringSubmatch(string(page), -1)
    mut.Lock()
    for _, element := range match {
        url := strings.Replace(strings.Replace(element[0], "<a href=", "", -1), ">", "", -1)
        trimmed := trimFromSpace(url)
        if !checkIfVisited(trimmed, crawler) {
            crawler.visited = append(crawler.visited, trimmed)
        }
    }
    mut.Unlock()
    if wg != nil {
        wg.Done()
    }
}

// See if URL was Already Found
func checkIfVisited(testString string, crawler *WebCrawler) bool{
    for _, url := range crawler.visited {
        if url == testString {
            return true
        }
    }
    return false
}

// Compare Two Slices
func checkEqual(tmp, crawled []string) bool{
    if len(tmp) != len(crawled) {
        return false
    }
    for index, val := range tmp{
        if val != crawled[index]{
            return false
        }
    }
    return true
}

// Trim Off Extra Stuff...
func trimFromSpace(stringToTrim string) string{
    if index := strings.Index(stringToTrim, " "); index != -1 {
        return stringToTrim[:index]
    }
    return stringToTrim
}
