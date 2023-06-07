package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/ido50/sqlz"
	_ "github.com/lib/pq"
	metis "github.com/metis-data/go-interceptor"
	"go.opentelemetry.io/otel"
)

type User struct {
	ID   int
	Name string
}

func main() {
	log.Printf("starting web server")

	// Create a new metis tracer provider
	tp, err := metis.NewTracerProvider()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()
	otel.SetTracerProvider(tp)

	// Create a new gorilla/mux router
	router := mux.NewRouter()
	router.HandleFunc("/", getRoot)

	// Wrap the router with the metis handler
	handler := metis.NewHandler(router, "http-server-gorilla-sqlz")

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

func getRoot(w http.ResponseWriter, r *http.Request) {
	dbHost := "postgres"
	dbPort := 5432
	dbUser := "postgres"
	dbPassword := "postgres"
	dbName := "my_database"

	dataSourceName := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// Open a connection to the database via metis API
	db, err := metis.OpenDB(dataSourceName)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	var user User
	err = sqlz.New(db, "postgres").
		Select("id", "name").
		From("my_schema.my_table").
		GetRow(&user)
	if err != nil {
		panic(err)
	}

	fmt.Printf("ID: %d, Name: %s\n", user.ID, user.Name)

	log.Printf("got / request\n")
	io.WriteString(w, "This is my website!\n")
}
