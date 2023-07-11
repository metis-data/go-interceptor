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
	_ "github.com/lib/pq"
	metis "github.com/metis-data/go-interceptor"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var tp *trace.TracerProvider

type User struct {
	ID   int
	Name string
}

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

	// Create a new gorilla/mux router
	router := mux.NewRouter()

	router.HandleFunc("/", getRoot)
	router.HandleFunc("/shutdown", shutdownHandler)

	// Wrap the router with the metis handler
	metisRouter, err := metis.WrapGorillaMuxRouter(router)
	if err != nil {
		log.Fatal(err)
	}
	handler := metis.NewHandler(metisRouter, "web-go-gorm")
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

	// Open a connection to the database via metis API
	db, err := metis.OpenDB(dataSourceName)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	gormDB = gormDB.WithContext(r.Context()) // make sure to pass the request context to GORM
	var users []User
	gormDB.Raw(fmt.Sprintf("SELECT id, name FROM %s.my_table", dbSchema)).Find(&users)
	for _, user := range users {
		fmt.Printf("ID: %d, Name: %s\n", user.ID, user.Name)
	}

	log.Printf("got / request\n")
	io.WriteString(w, "This is my website!\n")
}
