package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
}

var (
	rng   = rand.New(rand.NewSource(time.Now().UnixNano()))
	rngMu sync.Mutex
)

func randomUserAgent() string {
	rngMu.Lock()
	defer rngMu.Unlock()
	return userAgents[rng.Intn(len(userAgents))]
}

func getRequest(targetURL string) (*http.Response, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", randomUserAgent())

	return client.Do(req)
}

func discoverLinks(response *http.Response) []string {
	if response == nil {
		return nil
	}

	doc, err := goquery.NewDocumentFromResponse(response)
	if err != nil {
		return nil
	}

	foundURLs := []string{}
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		res, ok := s.Attr("href")
		if ok && res != "" {
			foundURLs = append(foundURLs, res)
		}
	})

	return foundURLs
}

func checkRelative(href string, baseURL string) string {
	if strings.HasPrefix(href, "/") {
		return fmt.Sprintf("%s%s", baseURL, href)
	}
	return href
}

func resolveRelativeLinks(href string, baseURL string) (bool, string) {
	if href == "" {
		return false, ""
	}

	resultHref := checkRelative(href, baseURL)
	baseParse, err := url.Parse(baseURL)
	if err != nil {
		return false, ""
	}
	resultParse, err := url.Parse(resultHref)
	if err != nil {
		return false, ""
	}

	if baseParse.Host == resultParse.Host {
		return true, resultHref
	}
	return false, ""
}

var tokens = make(chan struct{}, 5) // Semaphore to limit concurrent requests

func Crawl(targetURL string, baseURL string) []string {
	fmt.Println(targetURL)
	tokens <- struct{}{}
	resp, err := getRequest(targetURL)
	<-tokens

	if err != nil || resp == nil {
		return nil
	}
	defer resp.Body.Close()

	links := discoverLinks(resp)
	foundURLs := []string{}
	for _, link := range links {
		ok, correctLink := resolveRelativeLinks(link, baseURL)
		if ok && correctLink != "" {
			foundURLs = append(foundURLs, correctLink)
		}
	}

	ParseHTML(resp)
	return foundURLs
}

func ParseHTML(response *http.Response) {
	// Implement custom parsing here for the content you need
	_ = response
}

func main() {
	worklist := make(chan []string)
	var n int
	n++
	baseDomain := "https://www.theguardian.com"
	go func() { worklist <- []string{baseDomain} }()
	seen := make(map[string]bool)

	for ; n > 0; n-- {
		list := <-worklist
		for _, link := range list {
			if !seen[link] {
				seen[link] = true
				n++
				go func(link string, baseURL string) {
					foundLinks := Crawl(link, baseURL)
					if foundLinks != nil {
						worklist <- foundLinks
					}
				}(link, baseDomain)
			}
		}
	}
}
