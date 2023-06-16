package metis

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/LeonPev/otelsql"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// NewTracerProvider returns a new tracer provider with the metis exporter.
// The url and apiKey can be set with the environment variables METIS_EXPORTER_URL and METIS_API_KEY.
func NewTracerProvider() (*trace.TracerProvider, error) {
	url := os.Getenv("METIS_EXPORTER_URL")
	apiKey := os.Getenv("METIS_API_KEY")
	return NewTracerProviderWithLogin(url, apiKey)
}

// NewTracerProviderWithLogin returns a new tracer provider with the metis exporter.
func NewTracerProviderWithLogin(url, apiKey string) (*trace.TracerProvider, error) {
	if url == "" {
		url = "https://ingest.metisdata.io/"
	}
	if apiKey == "" {
		return nil, fmt.Errorf("METIS_API_KEY environment variable not set")
	}
	ms := newMetisServer(url, apiKey)
	exporter, err := newExporter(ms)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP trace exporter: %w", err)
	}
	// exporter, err := stdouttrace.New() // TODO: remove
	// if err != nil {
	// 	return nil, fmt.Errorf("creating OTLP trace exporter: %w", err)
	// }
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(newResource()),
	)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return tp, nil
}

// OpenDB returns a new wrapped sql.DB connection.
func OpenDB(dataSourceName string) (*sql.DB, error) {
	return otelsql.Open("postgres", dataSourceName, otelsql.WithAttributes(
		semconv.DBSystemPostgreSQL,
	), otelsql.WithSQLCommenter(true))
}

type dsnConnector struct {
	dsn    string
	driver driver.Driver
}

func (t dsnConnector) Connect(_ context.Context) (driver.Conn, error) {
	return t.driver.Open(t.dsn)
}

func (t dsnConnector) Driver() driver.Driver {
	return t.driver
}

// NewHandler returns a new http.Handler that instruments requests with OpenTelemetry.
func NewHandler(handler http.Handler, operation string) http.Handler {
	return otelhttp.NewHandler(handler, operation)
}

type metisServer struct {
	url    string
	apiKey string
	client *http.Client
}

func newMetisServer(url, apiKey string) *metisServer {
	return &metisServer{
		url:    url,
		apiKey: apiKey,
		client: &http.Client{},
	}
}

func (m *metisServer) Write(p []byte) (n int, err error) {
	req, err := http.NewRequest("POST", m.url, bytes.NewReader(p))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", m.apiKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return len(p), nil
}

// newExporter returns a console exporter.
func newExporter(w io.Writer) (trace.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithWriter(w),
		stdouttrace.WithoutTimestamps(),
	)
}

// newResource returns a resource describing this application.
func newResource() *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("metis-go-client"),
			semconv.ServiceVersion("v0.1.0"),
			attribute.String("environment", "demo"),
		),
	)
	return r
}
