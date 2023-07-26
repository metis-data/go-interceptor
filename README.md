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
```shell
go get github.com/metis-data/go-interceptor \
  go.opentelemetry.io/otel \
  go.opentelemetry.io/otel/sdk/trace
```

- Set the api key environment variable: ```METIS_API_KEY```

    Metis Api Key can be generated at [Metis](https://app.metisdata.io/)

- Enable Otel instrumentation:
  1. Set up Tracer:
  ```go
  import (
    metis "github.com/metis-data/go-interceptor"
    _ "github.com/lib/pq"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/sdk/trace"
  )
  
  var tp *trace.TracerProvider
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
  
  import (
    "fmt"
    "net/http"
    
    metis "github.com/metis-data/go-interceptor"
    _ "github.com/lib/pq"
  )
  
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
  
  import (
    "fmt"
    "net/http"
    
    "github.com/gorilla/mux"
    metis "github.com/metis-data/go-interceptor"
    _ "github.com/lib/pq"
  )
  
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

  import (
    "database/sql"
    "fmt"
    "log"

    metis "github.com/metis-data/go-interceptor"
    _ "github.com/lib/pq"
  )

  dbHost := "postgres"
  dbPort := 5432
  dbUser := "postgres"
  dbPassword := "postgres"
  dbName := "my_database"
  dbSchema := "my_schema"

  dataSourceName := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
      dbHost, dbPort, dbUser, dbPassword, dbName)

  var db *sql.DB
  var err error
  
  // Open a connection to the database via metis API
  db, err = metis.OpenDB(dataSourceName)
  if err != nil {
      log.Fatal(err)
  }
  defer db.Close()
  ```
  4. Pass context in queries:
  ```go
  // lib/pq
  
  import (
    "database/sql"
    "fmt"
    "log"
    "net/http"
  )
  
  query := fmt.Sprintf("SELECT id, name FROM %s.my_table", dbSchema)
  
  var rows *sql.Rows
  var err error
  
  // make sure to pass the context here
  // r *http.Request
  rows, err = db.QueryContext(r.Context(), query)
  if err != nil {
      log.Fatal(err)
  }
  defer rows.Close()
  ```
  ```go
  // gorm
  
  import (
    "database/sql"
    "fmt"
    "log"
    "net/http"
    
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
  )
  
  query := fmt.Sprintf("SELECT id, name FROM %s.my_table", dbSchema)
  
  var gormDB *gorm.DB
  var err error
  
  gormDB, err = gorm.Open(postgres.New(postgres.Config{
      Conn: db,
  }), &gorm.Config{})
  if err != nil {
      log.Fatal(err)
  }
  
  // make sure to pass the context here
  // r *http.Request
  gormDB = gormDB.WithContext(r.Context())
  var users []User
  gormDB.Raw(query).Find(&users)
  ```
  ```go
  // ido50/sqlz
  
  import (
    "database/sql"
    "fmt"
    "log"
    "net/http"
    
    "github.com/ido50/sqlz"
  )

  var sqlzDB *sqlz.DB
  var user User
  var err error
  
  sqlzDB = sqlz.New(db, "postgres")

  // make sure to pass the request context here
  // r *http.Request
  err = sqlzDB.
  Select("id", "name").
  From("my_schema.my_table").
  GetRowContext(r.Context(), &user)
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