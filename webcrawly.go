package main

import (
	"fmt"
	"net/http"
	"io"
	"log"
	"sync"
	"golang.org/x/net/html"
	"net/url"
)

// fetch function to fetch html content of the url

func fetch( url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// parse function to parse the fetched html to extract links
func parseLinks(body string) ([]string, error) {
	var links []string
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					links = append(links, a.Val)
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return links, nil
}

// url resolution function to ensure that relative urls are correctly
// converted to absolute urls
func resolveURL(link, base string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", err
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	resolvedURL := baseURL.ResolveReference(u)
	return resolvedURL.String(), nil
}

// worker function that will process urls concurrently
func worker(urls chan string, wg *sync.WaitGroup, visited *sync.Map) {
	defer wg.Done()
	for url := range urls {
		// check if url bas been visited
		if _, ok := visited.Load(url); ok {
			continue
		}
		visited.Store(url, true)

		body, err := fetch(url)
		if err != nil {
			log.Printf("Failed to fetch %s: %v", url, err)
			continue
		}

		links, err := parseLinks(body)
		if err != nil {
			log.Printf("Failed to parse links on %s: %v", url, err)
			continue
		}

		for _, link := range links {
			absURL, err := resolveURL(link, url)
			if err != nil {
				continue
			}
			// enqueue the url for crawling
			urls <- absURL
		}
	}
}

// main function
func main() {
	startURL := ""
	urls := make(chan string)
	var wg sync.WaitGroup
	visited := &sync.Map{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go worker(urls, &wg, visited)
	}

	urls <- startURL

	go func() {
		wg.Wait()
		close(urls)
	}()

	select {}
}