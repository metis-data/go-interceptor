package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("io.ReadAll() error = %v", err)
		}
		bodyStr := string(body)
		log.Println(bodyStr)
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
		time.Sleep(2 * time.Second)
		// send shutdown request
		for _, url := range urls {
			log.Printf("sending shutdown request to %s", url)
			resp, err := http.Get(fmt.Sprintf("%s/shutdown", url))
			if err != nil {
				log.Printf("http.Get() error = %v", err)
			}
			resp.Body.Close()
		}
	}(urls)

	recivedSpans := []string{}

	for {
		select {
		case <-time.After(60 * time.Second):
			log.Printf("timeout")
			os.Exit(1)
		case span := <-spans:
			log.Println("recived span")
			recivedSpans = append(recivedSpans, span)
			if len(recivedSpans) == len(urls) {
				os.Exit(0)
			}
		}
	}
}
