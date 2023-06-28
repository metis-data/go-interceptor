package metis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
)

type metisMockServer struct {
	spans []string
	t     *testing.T
}

func (m *metisMockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bodyStr, err := io.ReadAll(r.Body)
	if err != nil {
		m.t.Errorf("io.ReadAll() error = %v", err)
	}
	m.spans = append(m.spans, string(bodyStr))
	w.WriteHeader(http.StatusOK)
}

func TestNewTracerProvider(t *testing.T) {
	mm := &metisMockServer{t: t}
	// setup mock metis server
	ts := httptest.NewServer(http.HandlerFunc(mm.ServeHTTP))
	defer ts.Close()

	// setup telemetry
	tp, err := NewTracerProviderWithLogin(ts.URL, "test-api-key")
	if err != nil {
		t.Errorf("NewTracerProvider() error = %v", err)
	}
	otel.SetTracerProvider(tp)

	// send a trace
	_, span := otel.Tracer("balagan").Start(context.Background(), "gadol")
	span.End()

	// flush all spans
	if err := tp.Shutdown(context.Background()); err != nil {
		t.Errorf("tp.Shutdown() error = %v", err)
	}
	if !strings.Contains(mm.spans[0], "balagan") || !strings.Contains(mm.spans[0], "gadol") {
		t.Errorf("expected span to contain balagan gadol")
	}
}
func TestTraceProviderBatcherWithByteSizeLimit(t *testing.T) {
	queueSize = 10000
	mm := &metisMockServer{t: t}

	// setup mock metis server
	ts := httptest.NewServer(http.HandlerFunc(mm.ServeHTTP))
	defer ts.Close()

	// setup telemetry
	tp, err := NewTracerProviderWithLogin(ts.URL, "test-api-key")
	if err != nil {
		t.Errorf("NewTracerProvider() error = %v", err)
	}
	otel.SetTracerProvider(tp)

	// send traces to fill up the batcher
	for i := 0; i < 10; i++ {
		_, span := otel.Tracer(fmt.Sprintf("test-%d", i)).Start(context.Background(), "Run")
		span.End()
	}

	// flush all spans
	if err := tp.Shutdown(context.Background()); err != nil {
		t.Fatalf("tp.Shutdown() error = %v", err)
	}

	if len(mm.spans) != 2 {
		t.Fatalf("expected 2 span chanks, got %d", len(mm.spans))
	}
	if len(mm.spans[0]) != 9433 {
		t.Errorf("expected 9433 span, got %d", len(mm.spans[0]))
	}
	if len(mm.spans[1]) != 1049 {
		t.Errorf("expected 1049 span, got %d", len(mm.spans[1]))
	}
	// check all spans where processed
	for i := 0; i < 9; i++ {
		if !strings.Contains(mm.spans[0], fmt.Sprintf("test-%d", i)) {
			t.Errorf("expected span %d to be in the first batch", i)
		}
	}
	if !strings.Contains(mm.spans[1], "test-9") {
		t.Errorf("expected span 9 to be in the second batch")
	}
	// check if the spanc are valid json
	if err := validateJSON(mm.spans[0]); err != nil {
		t.Errorf("validateJSON() error = %v", err)
	}
	if err := validateJSON(mm.spans[1]); err != nil {
		t.Errorf("validateJSON() error = %v", err)
	}
}

func validateJSON(s string) error {
	var js []map[string]interface{}
	if err := json.Unmarshal([]byte(s), &js); err != nil {
		return err
	}
	return nil
}
