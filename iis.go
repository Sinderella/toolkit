package main

import (
	"bufio"
	"fmt"
	"log"
	net_url "net/url"
	"os"
	"strings"
	"sync"

	req "github.com/levigross/grequests"
)

var F, _ = os.OpenFile("debug.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

type URLRecord struct {
	URL string
	IIS string
}

func NewURLRecord(url string) *URLRecord {
	return &URLRecord{url, ""}
}

func getRequestOptions() *req.RequestOptions {
	/*
		proxies := make(map[string]*url.URL)
		u, err := url.Parse("http://127.0.0.1:8080")
		if err != nil {
			log.Fatal(err)
		}
		proxies["http"] = u
		proxies["https"] = u
		return &req.RequestOptions{InsecureSkipVerify: true, Proxies: proxies}
	*/
	return &req.RequestOptions{InsecureSkipVerify: true}
}

func ScanURL(done <-chan struct{}, urls <-chan string, result chan<- *URLRecord) {
	for url := range urls {
		fmt.Printf("Scanning %s\n", url)
		if !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
			url = "http://" + url
		}
		ur := NewURLRecord(url)

		resp, err := req.Get(ur.URL, getRequestOptions())
		if err != nil {
			switch err.(type) {
			case *net_url.Error:
				ur.URL = "https" + ur.URL[4:len(ur.URL)]
				resp, err = req.Get(ur.URL, getRequestOptions())
				if err != nil {
					log.Println("Website unreachable: ", err)
					return
				}
			default:
				log.Println("Website unreachable: ", err)
				return
			}
		}

		// TODO: detect IIS version
		for key, value := range resp.Header {
			if hasElem(key, HeaderKeys) {
				ur.Headers[key] = strings.Join(value, " ")
			}
		}

		for _, key := range HeaderKeys {
			if !hasKey(key, ur.Headers) {
				ur.MissingHeaders = append(ur.MissingHeaders, key)
			}
		}

		select {
		case result <- ur:
		case <-done:
			return
		}
	}
}

func fetchURLs(done <-chan struct{}, urls <-chan string) (<-chan *URLRecord, <-chan error) {
	ur := make(chan *URLRecord)
	errc := make(chan error, 1)

	var wg sync.WaitGroup
	const numFetchers = 4
	wg.Add(numFetchers)
	for i := 0; i < numFetchers; i++ {
		go func() {
			ScanURL(done, urls, ur)
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(ur)
	}()

	return ur, errc
}

func FetchAll(filename string) (URLRecords, error) {
	done := make(chan struct{})
	urls := make(chan string)
	defer close(done)

	c, _ := fetchURLs(done, urls)

	log.Printf("Reading %s...", filename)

	file, err := os.Open(filename)

	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	go func() {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			urls <- scanner.Text()
		}
		close(urls)
	}()

	m := make(URLRecords, 0)
	log.Printf("Waiting for result...\n")
	for r := range c {
		m = append(m, r)
	}

	return m, nil
}

func main() {
	log.SetOutput(F)

	if len(os.Args) != 2 {
		fmt.Println("It requires a filename to read")
		os.Exit(1)
	}

	filename := os.Args[1]
	results, _ := FetchAll(filename)

	fmt.Println("Grouping...")
	grouped := make(URLRecords, 0)
	for _, ur1 := range results {
		if grouped.isIn(ur1) {
			continue
		}
		grouping := make(URLRecords, 0)
		grouped = append(grouped, ur1)
		grouping = append(grouping, ur1)
		for _, ur2 := range results {
			if len(ur2.MissingHeaders) != len(ur1.MissingHeaders) || grouped.isIn(ur2) {
				continue
			} else if ur1.HasSameMissingHeaders(ur2) {
				grouping = append(grouping, ur2)
				grouped = append(grouped, ur2)
			}
		}

		for _, ur := range grouping {
			fmt.Printf("[%s]()\n", ur.URL)
		}

		for _, missingheader := range ur1.MissingHeaders {
			fmt.Printf(" * %s\n", missingheader)
		}
		fmt.Println()
	}
}
