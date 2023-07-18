[![metis](https://static-asserts-public.s3.eu-central-1.amazonaws.com/metis-min-logo.png)](https://www.metisdata.io/)

# Metis GO Interceptor

[Documentation](https://docs.metisdata.io)

## Supported GO packages
- http:
  1. net/http
  2. gorilla/mux
- postgres:
  1. lib/pq
  2. gorm
  3. ido50/sqlz

## Usage
- Run 
```
go get github.com/metis-data/go-interceptor \
  go.opentelemetry.io/otel \
  go.opentelemetry.io/otel/sdk/trace \
```

- Set the api key environment variable: ```METIS_API_KEY```

    Metis Api Key can be generated at [Metis](https://app.metisdata.io/)

- Enable Otel instrumentation:
  1. Set up Tracer:
  ```go
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
  ```
  2. Wrap your http server with metis:
  ```go	
  // net/http
  // use metis.NewServeMux() instead of http.NewServeMux()
  mux := metis.NewServeMux() 
  mux.HandleFunc("/api/endpoint", someHandler)
  ...
  
  // Wrap the router with the metis handler
  handler := metis.NewHandler(mux, "my-web-service")
  err = http.ListenAndServe(fmt.Sprintf(":%s", port), handler)
  ```
  ```go
  // gorilla/mux
  // Create a new gorilla/mux router
  router := mux.NewRouter()

  router.HandleFunc("/api/endpoint", someHandler)
  ...

  // Wrap the router with the metis handler
  metisRouter, err := metis.WrapGorillaMuxRouter(router)
  if err != nil {
      log.Fatal(err)
  }
  handler := metis.NewHandler(metisRouter, "web-go-gorm")
  err = http.ListenAndServe(fmt.Sprintf(":%s", port), handler)
  ```
  3. Wrap your database connection with metis:
  ```go
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
  ```
  4. Pass context in queries:
  ```go
  // lib/pq
  query := fmt.Sprintf("SELECT id, name FROM %s.my_table", dbSchema)
  
  // make sure to pass the context here
  rows, err := db.QueryContext(r.Context(), query)
  if err != nil {
      log.Fatal(err)
  }
  defer rows.Close()
  ```
  ```go
  // gorm
  query := fmt.Sprintf("SELECT id, name FROM %s.my_table", dbSchema)
  
  gormDB, err := gorm.Open(postgres.New(postgres.Config{
      Conn: db,
  }), &gorm.Config{})
  if err != nil {
      log.Fatal(err)
  }
  
  // make sure to pass the context here
  gormDB = gormDB.WithContext(r.Context())
  var users []User
  gormDB.Raw(query).Find(&users)
  ```
  ```go
  // ido50/sqlz
  var user User
  err = sqlz.New(db, "postgres").
      Select("id", "name").
      From("my_schema.my_table").
      GetRowContext(r.Context(), &user) // make sure to pass the context here
  if err != nil {
      panic(err)
  }
  ```

## Examples

- [net/http + lib/pq](https://github.com/metis-data/go-interceptor/blob/main/e2e/web/main.go)
- [gorilla/mux + gorm](https://github.com/metis-data/go-interceptor/blob/main/e2e/web-gorilla-gorm/main.go)
- [gorilla/mux + ido50/sqlz](https://github.com/metis-data/go-interceptor/blob/main/e2e/web-gorilla-sqlz/main.go)

## Issues
If you would like to report a potential issue please use [Issues](https://github.com/metis-data/go-interceptor/issues)

## License Summary
This code is made available under the MIT license.