package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	time.Sleep(5 * time.Second) // wait for postgres to start
	log.Printf("starting collector server")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	spans := make(chan string)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("ioutil.ReadAll() error = %v", err)
		}
		bodyStr := string(body)
		spans <- bodyStr
		w.WriteHeader(http.StatusOK)
	})

	log.Printf("Listening on port %s", port)
	go func(port string) {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
	}(port)

	// send requests to urls
	urls := []string{
		os.Getenv("DST_WEB"),
		os.Getenv("DST_WEB_GORILLA_GORM"),
		os.Getenv("DST_WEB_GORILLA_SQLZ"),
	}
	go func(urls []string) {
		for _, url := range urls {
			log.Printf("sending request to %s", url)
			resp, err := http.Get(url)
			if err != nil {
				log.Printf("http.Get() error = %v", err)
			}
			resp.Body.Close()
		}
	}(urls)

	recivedSpans := []string{}

	for {
		select {
		case <-time.After(30 * time.Second):
			log.Printf("timeout")
			os.Exit(1)
		case span := <-spans:
			log.Println("recived span")
			recivedSpans = append(recivedSpans, span)
			if len(recivedSpans) == 10 {
				if sqlQueryIsCorrect(recivedSpans, len(urls)) {
					os.Exit(0)
				} else {
					fmt.Println("sql query is not in spans")
					fmt.Println(recivedSpans)
					os.Exit(2)
				}
			}
		}
	}
}

func sqlQueryIsCorrect(spans []string, expected int) bool {
	count := 0
	for _, span := range spans {
		if strings.Contains(span, "SELECT id, name FROM my_schema.my_table") {
			count++
		}
	}
	return count == expected
}
