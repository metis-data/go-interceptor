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

	"github.com/gorilla/mux"
	"github.com/ido50/sqlz"
	_ "github.com/lib/pq"
	metis "github.com/metis-data/go-interceptor"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

var tp *trace.TracerProvider

type User struct {
	ID   int
	Name string
}

func main() {
	log.Printf("starting web server")

	// Create a new metis tracer provider
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

	// Create a new gorilla/mux router
	router := mux.NewRouter()
	router.HandleFunc("/", metis.WrapHandlerFunc(getRoot, "/"))                         // Wrap each handler with the metis handler
	router.HandleFunc("/shutdown", metis.WrapHandlerFunc(shutdownHandler, "/shutdown")) // Wrap each handler with the metis handler
	// Wrap the router with the metis handler
	handler := metis.NewHandler(router, "web-go-sqlz")

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

	dataSourceName := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var db *sql.DB
	var err error
	var sqlzDB *sqlz.DB

	// Open a connection to the database via metis API
	db, err = metis.OpenDB(dataSourceName)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	var user User
	sqlzDB = sqlz.New(db, "postgres")

	// make sure to pass the request context
	// r *http.Request
	err = sqlzDB.
		Select("id", "name").
		From("my_schema.my_table").
		GetRowContext(r.Context(), &user)
	if err != nil {
		panic(err)
	}

	fmt.Printf("ID: %d, Name: %s\n", user.ID, user.Name)

	log.Printf("got / request\n")
	io.WriteString(w, "This is my website!\n")
}
