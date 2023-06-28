package metis

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.opentelemetry.io/otel"
)

type metisMockServer struct {
	spans []string
	t     *testing.T
}

func (m *metisMockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bodyStr, err := ioutil.ReadAll(r.Body)
	if err != nil {
		m.t.Errorf("ioutil.ReadAll() error = %v", err)
	}
	m.spans = append(m.spans, string(bodyStr))
	w.WriteHeader(http.StatusOK)
}

func TestNewTracerProvider(t *testing.T) {
	expected := `{"Name":"Run","SpanContext":{"TraceID":"214728fa3f7ef3b60ca74536b1a46c21","SpanID":"fbf70507fc1e3763","TraceFlags":"01","TraceState":"","Remote":false},"Parent":{"TraceID":"00000000000000000000000000000000","SpanID":"0000000000000000","TraceFlags":"00","TraceState":"","Remote":false},"SpanKind":1,"StartTime":"0001-01-01T00:00:00Z","EndTime":"0001-01-01T00:00:00Z","Attributes":null,"Events":null,"Links":null,"Status":{"Code":"Unset","Description":""},"DroppedAttributes":0,"DroppedEvents":0,"DroppedLinks":0,"ChildSpanCount":0,"Resource":[{"Key":"environment","Value":{"Type":"STRING","Value":"demo"}},{"Key":"service.name","Value":{"Type":"STRING","Value":"metis-go-client"}},{"Key":"service.version","Value":{"Type":"STRING","Value":"v0.1.0"}},{"Key":"telemetry.sdk.language","Value":{"Type":"STRING","Value":"go"}},{"Key":"telemetry.sdk.name","Value":{"Type":"STRING","Value":"opentelemetry"}},{"Key":"telemetry.sdk.version","Value":{"Type":"STRING","Value":"1.16.0"}}],"InstrumentationLibrary":{"Name":"test","Version":"","SchemaURL":""}}
`
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
	_, span := otel.Tracer("test").Start(context.Background(), "Run")
	span.End()

	// flush all spans
	if err := tp.Shutdown(context.Background()); err != nil {
		t.Errorf("tp.Shutdown() error = %v", err)
	}
	if diff := cmp.Diff(expected[100:], mm.spans[0][100:]); diff != "" {
		t.Errorf("MakeGatewayInfo() mismatch (-want +got):\n%s", diff)
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
