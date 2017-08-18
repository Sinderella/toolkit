package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	req "github.com/levigross/grequests"
)

var re = regexp.MustCompile("<title>(.*)</title>")

// URLRecord is for keeping information for each URL
type URLRecord struct {
	URL       string
	IIS       string
	IsIIS     bool
	Reachable bool
}

// URLRecords is a slice of URLRecord
type URLRecords []*URLRecord

// NewURLRecord is for creating a new URLRecord
func NewURLRecord(url string) *URLRecord {
	return &URLRecord{url, "", false, true}
}

func getRequestOptions() *req.RequestOptions {
	/*
		proxies := make(map[string]*url.URL)
		u, err := url.Parse("http://127.0.0.1:8080")
		if err != nil {
			fmt.Fatal(err)
		}
		proxies["http"] = u
		proxies["https"] = u
		return &req.RequestOptions{InsecureSkipVerify: true, Proxies: proxies}
	*/
	return &req.RequestOptions{InsecureSkipVerify: true}
}

// ScanURL handles passed URLs to start collecting information and send each URLRecord through the result channel
func ScanURL(done <-chan struct{}, urls <-chan string, result chan<- *URLRecord) {
	for url := range urls {
		fmt.Printf("Scanning %s\n", url)
		if !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
			url = "http://" + url
		}
		ur := NewURLRecord(url)

		resp, err := req.Get(ur.URL, getRequestOptions())
		if err != nil {
			fmt.Println("Website unreachable: ", err)
			ur.Reachable = false
			result <- ur
		}

		body := resp.String()
		found := re.FindStringSubmatch(body)

		if len(found) < 2 {
			ur.IIS = "Not IIS? " + ur.URL
			ur.IsIIS = false
		} else {
			switch found[1] {
			case "IIS7":
				ur.IIS = "7"
				ur.IsIIS = true
			case "Microsoft Internet Information Services 8":
				ur.IIS = "8"
				ur.IsIIS = true
			case "IIS Windows Server":
				ur.IIS = "8.5"
				ur.IsIIS = true
			default:
				ur.IIS = "Not IIS? " + ur.URL
				ur.IsIIS = false
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

// FetchAll reads URLs off a file and then pass them to fetchURLs
func FetchAll(filename string) (URLRecords, error) {
	done := make(chan struct{})
	urls := make(chan string)
	defer close(done)

	c, _ := fetchURLs(done, urls)

	fmt.Printf("Reading %s...\n", filename)

	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: ", err)
	}
	defer file.Close()

	go func() {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			urls <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "error: ", err)
		}
		close(urls)
	}()

	m := make(URLRecords, 0)
	fmt.Printf("Waiting for result...\n")
	for r := range c {
		m = append(m, r)
	}

	return m, nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("It requires a filename to read")
		os.Exit(1)
	}

	filename := os.Args[1]
	results, _ := FetchAll(filename)

	fmt.Println("IIS")
	for _, result := range results {
		if result.IsIIS {
			fmt.Printf("%s\n", result.URL)
		}
	}

	fmt.Println("Non IIS")
	for _, result := range results {
		if !result.IsIIS {
			fmt.Printf("%s\n", result.URL)
		}
	}
	fmt.Println("Unreachable")
	for _, result := range results {
		if !result.Reachable {
			fmt.Printf("%s\n", result.URL)
		}
	
	/*
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
	*/
}
