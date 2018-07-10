package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ugorji/go/codec"
	"sourcegraph.com/sourcegraph/appdash"
	"sourcegraph.com/sourcegraph/appdash/httptrace"
	"sourcegraph.com/sourcegraph/appdash/sqltrace"
)

func datadogTracesToAppdashThings(in []datadogSpan) (appdash.SpanID, []appdash.Event) {
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

func main() {
	collector := appdash.NewRemoteCollector("localhost:7701")

	http.HandleFunc("/v0.3/traces", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			log.Printf("Received %q on /v0.3/traces path, expected a POST", r.Method)
			http.Error(w, "was expecting POST", http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Content-Type") != "application/msgpack" {
			log.Printf("Received Content-Type of %q on /v0.3/traces path, expected application/msgpack", r.Header.Get("Content-Type"))
			http.Error(w, "unexpected content type", http.StatusBadRequest)
			return
		}

		// Datadog-Meta-Lang-Version: 2.4.4
		// Datadog-Meta-Tracer-Version: 0.11.2
		// X-Datadog-Trace-Count: 1
		// Datadog-Meta-Lang: ruby
		// Datadog-Meta-Lang-Interpreter: ruby-x86_64-linux

		msgpack := codec.MsgpackHandle{
			RawToString: true,
		}
		decoder := codec.NewDecoder(r.Body, &msgpack)
		var traces [][]datadogSpan
		if err := decoder.Decode(&traces); err != nil {
			log.Print("error decoding msgpack: " + err.Error())
			http.Error(w, "error decoding msgpack", http.StatusBadRequest)
			return
		}

		eventsSent := 0

		for _, trace := range traces {
			spanID, events := datadogTracesToAppdashThings(trace)
			rec := appdash.NewRecorder(spanID, collector)
			rec.Name("This is a name")
			for _, e := range events {
				eventsSent++
				rec.Event(e)
			}
			rec.Finish()
		}
		log.Printf("Found %d batches, sent %d events", len(traces), eventsSent)
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.String())
		for k, vs := range r.Header {
			for _, v := range vs {
				log.Printf("  %s: %s", k, v)
			}
		}
		if b, err := ioutil.ReadAll(r.Body); err != nil {
			log.Print("Error reading body: " + err.Error())
		} else if len(b) > 0 {
			log.Print(">> " + string(b))
		}

		w.WriteHeader(200)
	})

	http.ListenAndServe(":12345", nil)
}
