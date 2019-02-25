package main

import (
	"log"
	"net/url"
	"strconv"
	"time"

	"sourcegraph.com/sourcegraph/appdash"
	"sourcegraph.com/sourcegraph/appdash/httptrace"
	"sourcegraph.com/sourcegraph/appdash/sqltrace"
)

type AppdashExporter struct {
	collector appdash.Collector
}

func NewRemoteAppdash(address string) *AppdashExporter {
	return &AppdashExporter{
		collector: appdash.NewRemoteCollector(address),
	}
}

func (e *AppdashExporter) Export(spans []datadogSpan) {
	spanID, events := e.convert(spans)
	rec := appdash.NewRecorder(spanID, e.collector)
	rec.Name("This is a name")
	for _, e := range events {
		rec.Event(e)
	}
	rec.Finish()
}

func (e *AppdashExporter) Flush() {

}

func (e *AppdashExporter) convert(in []datadogSpan) (appdash.SpanID, []appdash.Event) {
	spanID := appdash.SpanID{}
	var events []appdash.Event
	for _, s := range in {

		log.Print("Found span of type " + s.Type)

		if s.Type == "http" {

			spanID.Parent = appdash.ID(s.ParentID)
			spanID.Trace = appdash.ID(s.TraceID)
			spanID.Span = appdash.ID(s.SpanID)

			e := httptrace.ServerEvent{}
			u, err := url.Parse(s.Meta["http.base_url"])
			if err != nil {
				e.Request.Host = u.Host
			}
			e.Request.Method = s.Meta["http.method"]
			e.Request.URI = s.Meta["http.url"]
			e.Request.Proto = "HTTP/1.1"

			e.Response.StatusCode, _ = strconv.Atoi(s.Meta["http.status_code"])

			e.ServerRecv = time.Unix(0, int64(s.Start))
			e.ServerSend = time.Unix(0, int64(s.Start+s.Duration))
			events = append(events, e)
		} else if s.Type == "sql" {
			e := sqltrace.SQLEvent{
				SQL:        s.Resource,
				ClientSend: time.Unix(0, int64(s.Start)),
				ClientRecv: time.Unix(0, int64(s.Start+s.Duration)),
			}
			events = append(events, e)
		}
	}

	return spanID, events
}
