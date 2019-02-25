package main

import (
	"encoding/binary"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
)

type jaegerExporter struct {
	jaeger *jaeger.Exporter
}

func NewJaegerExporter(serviceName string, collector string, agent string) (*jaegerExporter, error) {
	exporter, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint:     agent,
		CollectorEndpoint: collector,
		Process: jaeger.Process{
			ServiceName: serviceName,
		},
	})
	if err != nil {
		return nil, err
	}
	return &jaegerExporter{
		jaeger: exporter,
	}, nil
}

func (e *jaegerExporter) Export(in []datadogSpan) {
	for _, datadogSpan := range in {
		e.exportDatadogSpan(datadogSpan)
	}
}

func (e *jaegerExporter) Flush() {
	e.jaeger.Flush()
}

func (e *jaegerExporter) exportDatadogSpan(in datadogSpan) {
	s := trace.SpanData{
		SpanContext: trace.SpanContext{
			TraceID: uint64To16Bytes(in.TraceID),
			SpanID:  uint64ToBytes(in.SpanID),
		},
		ParentSpanID: uint64ToBytes(in.ParentID),
		StartTime:    time.Unix(0, int64(in.Start)),
		EndTime:      time.Unix(0, int64(in.Start+in.Duration)),
		Attributes:   map[string]interface{}{},
	}

	switch in.Type {
	case "http":
		e.fillInHTTP(in, &s)
	case "sql":
		e.fillInSQL(in, &s)
	default:
		e.fillInGeneric(in, &s)
	}

	e.jaeger.ExportSpan(&s)
}

func (e *jaegerExporter) fillInHTTP(in datadogSpan, s *trace.SpanData) {
	switch in.Name {
	case "rack.request":
		s.Name = "Recv." + in.Meta["http.url"]
		u, _ := url.Parse(in.Meta["http.base_url"])
		s.Attributes["http.host"] = u.Host
		s.Attributes["http.method"] = in.Meta["http.method"]
		s.Attributes["http.path"] = in.Meta["http.url"]
		// s.Attributes["http.route"] = "/users/:userID"
		// I haven't seen user agents come through from dd-trace-rb
		// s.Attributes["http.user_agent"] = in.Meta["http.response.headers.user_agent"]
		statusCode, _ := strconv.Atoi(in.Meta["http.status_code"])
		s.Attributes["http.status_code"] = int64(statusCode)
		s.Attributes["http.url"] = in.Meta["http.base_url"] + in.Meta["http.url"]

		// https://github.com/census-instrumentation/opencensus-specs/blob/master/trace/HTTP.md#mapping-from-http-status-codes-to-trace-status-codes
		s.Status = ochttp.TraceStatus(statusCode, http.StatusText(statusCode))
	default:
		// rails.action_controller
		e.fillInGeneric(in, s)
	}
}

func (e *jaegerExporter) fillInSQL(in datadogSpan, s *trace.SpanData) {
	s.Attributes["sql.query"] = in.Resource
	e.fillInGeneric(in, s)
}

func (*jaegerExporter) fillInGeneric(in datadogSpan, s *trace.SpanData) {
	s.Name = in.Name
	for key, value := range in.Meta {
		s.Attributes[key] = value
	}
	if in.Error > 0 {
		s.Status.Code = trace.StatusCodeUnknown
		s.Status.Message = "something bad happened"
	}
}

func uint64ToBytes(i uint64) [8]byte {
	var out [8]byte
	binary.BigEndian.PutUint64(out[:], i)
	return out
}

func uint64To16Bytes(i uint64) [16]byte {
	var out [16]byte
	binary.BigEndian.PutUint64(out[8:], i)
	return out
}
