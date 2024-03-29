package main

import (
	"context"
	"database/sql"
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

	mux := metis.NewServeMux() // use metis.NewServeMux() instead of http.NewServeMux()
	mux.HandleFunc("/", getRoot)
	mux.HandleFunc("/shutdown", shutdownHandler)
	// Wrap the router with the metis handler
	handler := metis.NewHandler(mux, "my-web-service")

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
	log.Print("Shutdown Server")
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

	var db *sql.DB
	var rows *sql.Rows
	var err error

	// Open a connection to the database via metis api
	db, err = metis.OpenDB(dataSourceName)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	query := fmt.Sprintf("SELECT id, name FROM %s.my_table", dbSchema)

	// make sure to pass the context here
	// r *http.Request
	rows, err = db.QueryContext(r.Context(), query)
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
