package metis

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/LeonPev/otelsql"
	"github.com/getsentry/sentry-go"
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
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              "https://d3d9fcb6cf4041a6a085dafd56b80ef8@o1173646.ingest.sentry.io/6268970",
		TracesSampleRate: 1.0,
	})
	if err != nil {
		return nil, fmt.Errorf("sentry.Init() error = %w", err)
	}
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetContext("Details", map[string]interface{}{
			"'Api Key'": apiKey,
		})
	})
	if url == "" {
		url = "https://ingest.metisdata.io/"
	}
	if apiKey == "" {
		return nil, fmt.Errorf("METIS_API_KEY environment variable not set")
	}
	exporter, err := newMetisExporter(url, apiKey)
	if err != nil {
		sentry.CaptureException(err)
		return nil, fmt.Errorf("creating OTLP trace exporter: %w", err)
	}
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
		trace.WithResource(newResource()),
	)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return tp, nil
}

type metisExporter struct {
	ms             *metisServer
	loader         *spanLoader
	loadExp        trace.SpanExporter
	queue          []trace.ReadOnlySpan
	queueBytesSize int
}

var queueSize = 150000 // 150000 bytes

func newMetisExporter(url, apiKey string) (*metisExporter, error) {
	ms := &metisServer{
		url:    url,
		apiKey: apiKey,
		client: &http.Client{},
	}
	loader := &spanLoader{}
	loadExp, err := stdouttrace.New(
		stdouttrace.WithWriter(loader),
		stdouttrace.WithoutTimestamps(),
	)
	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}
	return &metisExporter{
		ms:             ms,
		loadExp:        loadExp,
		loader:         loader,
		queue:          []trace.ReadOnlySpan{},
		queueBytesSize: 0,
	}, nil
}

func (m *metisExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	// export spans to metis server. With size limit of queueSize bytes.
	for _, span := range spans {
		size, err := m.getSpanSize(span)
		if err != nil {
			sentry.CaptureException(err)
			return err
		}
		if m.queueBytesSize+size > queueSize {
			err = m.exportQueue(ctx)
			if err != nil {
				sentry.CaptureException(err)
				return err
			}
		}
		m.queue = append(m.queue, span)
		m.queueBytesSize += size
	}
	return nil
}

func (m *metisExporter) getSpanSize(span trace.ReadOnlySpan) (int, error) {
	err := m.loadExp.ExportSpans(context.Background(), []trace.ReadOnlySpan{span})
	if err != nil {
		sentry.CaptureException(err)
		return 0, err
	}
	return len(m.loader.spanText), nil
}

func (m *metisExporter) exportQueue(ctx context.Context) error {
	spansToExportBytes, err := m.convertQueueToJSON(ctx)
	if err != nil {
		sentry.CaptureException(err)
		return err
	}
	err = m.ms.Export(spansToExportBytes)
	if err != nil {
		sentry.CaptureException(err)
		return err
	}
	m.queue = []trace.ReadOnlySpan{}
	m.queueBytesSize = 0
	return nil
}

func (m *metisExporter) convertQueueToJSON(ctx context.Context) ([]byte, error) {
	var spans []string
	for _, span := range m.queue {
		err := m.loadExp.ExportSpans(ctx, []trace.ReadOnlySpan{span})
		if err != nil {
			sentry.CaptureException(err)
			return nil, err
		}
		spanText := strings.TrimRight(m.loader.spanText, "\n")
		spans = append(spans, spanText)
	}
	return convertJSONBytesSliceToSingleJSOM(spans)
}

func convertJSONBytesSliceToSingleJSOM(spans []string) ([]byte, error) {
	var res []map[string]interface{}
	for _, span := range spans {
		var spanMap map[string]interface{}
		err := json.Unmarshal([]byte(span), &spanMap)
		if err != nil {
			sentry.CaptureException(err)
			return nil, err
		}
		res = append(res, spanMap)
	}
	return json.Marshal(res)
}

func (m *metisExporter) Shutdown(ctx context.Context) error {
	return m.exportQueue(ctx)
}

type spanLoader struct {
	spanText string
}

func (s *spanLoader) Write(p []byte) (n int, err error) {
	s.spanText = string(p)
	return len(p), nil
}

type metisServer struct {
	url    string
	apiKey string
	client *http.Client
}

func (m *metisServer) Export(p []byte) error {
	req, err := http.NewRequest("POST", m.url, bytes.NewReader(p))
	if err != nil {
		sentry.CaptureException(err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", m.apiKey)

	resp, err := m.client.Do(req)
	if err != nil {
		sentry.CaptureException(err)
		return err
	}
	resp.Body.Close()
	return nil
}

// newExporter returns a console exporter.
// func newExporter(w io.Writer) (trace.SpanExporter, error) {
// 	return stdouttrace.New(
// 		stdouttrace.WithWriter(w),
// 		stdouttrace.WithoutTimestamps(),
// 	)
// }

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

// OpenDB returns a new wrapped sql.DB connection.
func OpenDB(dataSourceName string) (*sql.DB, error) {
	return otelsql.Open("postgres", dataSourceName, otelsql.WithAttributes(
		semconv.DBSystemPostgreSQL,
	), otelsql.WithSQLCommenter(true))
}

// NewHandler returns a new http.Handler that instruments requests with OpenTelemetry.
func NewHandler(handler http.Handler, operation string) http.Handler {
	return otelhttp.NewHandler(handler, operation)
}
