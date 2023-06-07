package metis

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.opentelemetry.io/otel"
)

func TestNewTracerProvider(t *testing.T) {
	spans := []string{}
	expected := `{"Name":"Run","SpanContext":{"TraceID":"214728fa3f7ef3b60ca74536b1a46c21","SpanID":"fbf70507fc1e3763","TraceFlags":"01","TraceState":"","Remote":false},"Parent":{"TraceID":"00000000000000000000000000000000","SpanID":"0000000000000000","TraceFlags":"00","TraceState":"","Remote":false},"SpanKind":1,"StartTime":"0001-01-01T00:00:00Z","EndTime":"0001-01-01T00:00:00Z","Attributes":null,"Events":null,"Links":null,"Status":{"Code":"Unset","Description":""},"DroppedAttributes":0,"DroppedEvents":0,"DroppedLinks":0,"ChildSpanCount":0,"Resource":[{"Key":"environment","Value":{"Type":"STRING","Value":"demo"}},{"Key":"service.name","Value":{"Type":"STRING","Value":"metis-go-client"}},{"Key":"service.version","Value":{"Type":"STRING","Value":"v0.1.0"}},{"Key":"telemetry.sdk.language","Value":{"Type":"STRING","Value":"go"}},{"Key":"telemetry.sdk.name","Value":{"Type":"STRING","Value":"opentelemetry"}},{"Key":"telemetry.sdk.version","Value":{"Type":"STRING","Value":"1.16.0"}}],"InstrumentationLibrary":{"Name":"test","Version":"","SchemaURL":""}}
`
	// setup mock metis server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyStr, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Errorf("ioutil.ReadAll() error = %v", err)
		}
		spans = append(spans, string(bodyStr))
		w.WriteHeader(http.StatusOK)
	}))
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
	if diff := cmp.Diff(expected[100:], spans[0][100:]); diff != "" {
		t.Errorf("MakeGatewayInfo() mismatch (-want +got):\n%s", diff)
	}
}
