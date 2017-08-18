package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "", nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: ", err)
		os.Exit(1)
	}

	req.Header.Set("Referer", "")

	for i := 0; i < 4000; i++ {
		fmt.Printf("[%d] requesting...\n", i+1)
		_, err := client.Do(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: ", err)
		}
	}
}
