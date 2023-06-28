package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	metis "github.com/metis-data/go-interceptor"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

var tp *trace.TracerProvider

func main() {
	log.Printf("starting web server")
	os.Setenv("METIS_API_KEY", "42") // TODO: remove
	// create a new metis tracer provider
	var err error
	tp, err = metis.NewTracerProvider()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()
	otel.SetTracerProvider(tp)

	mux := http.NewServeMux()
	mux.HandleFunc("/", getRoot)
	mux.HandleFunc("/shutdown", shutdownHandler)
	// wrap the mux with the metis handler
	handler := metis.NewHandler(http.HandlerFunc(mux.ServeHTTP), "http-server")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Listening on port %s\n", port)
	err = http.ListenAndServe(fmt.Sprintf(":%s", port), handler)
	if errors.Is(err, http.ErrServerClosed) {
		log.Printf("server closed\n")
	} else if err != nil {
		log.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}

func shutdownHandler(w http.ResponseWriter, r *http.Request) {
	if err := tp.Shutdown(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func getRoot(w http.ResponseWriter, r *http.Request) {
	dbHost := "postgres"
	dbPort := 5432
	dbUser := "postgres"
	dbPassword := "postgres"
	dbName := "my_database"
	dbSchema := "my_schema"

	dataSourceName := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// Open a connection to the database via metis api
	db, err := metis.OpenDB(dataSourceName)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	query := fmt.Sprintf("SELECT id, name FROM %s.my_table", dbSchema)
	rows, err := db.QueryContext(r.Context(), query) // make sure to pass the context here
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		err := rows.Scan(&id, &name)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("ID: %d, Name: %s\n", id, name)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	log.Printf("got / request\n")
	io.WriteString(w, "This is my website!\n")
}
