package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/ugorji/go/codec"
)

type exporter interface {
	Export(in []datadogSpan)
	Flush()
}

func main() {
	appdashAddress := flag.String("appdash", "", "Address of appdash collector to use (i.e. localhost:7701). If not specified, no appdash traces will be sent")
	exportDirectory := flag.String("dir", "", "Directory to record traces to. If not specified, traces won't be written to file")
	jaegerServiceName := flag.String("jaeger.service", "", "The service name to report to Jaeger as")
	jaegerCollectorAddress := flag.String("jaeger.collector", "", "The collector URL for Jaeger (e.g. http://jaeger.lan:14268/api/traces)")
	jaegerAgentAddress := flag.String("jaeger.agent", "", "The agent address for Jaeger (e.g. jaeger.lan:6831)")
	addr := flag.String("addr", ":12345", "The address to listen on for incoming Datadog traces")
	flag.Parse()

	var exporters []exporter

	if *appdashAddress != "" {
		exporters = append(exporters, NewRemoteAppdash(*appdashAddress))
		log.Print("Sending traces to appdash at " + *appdashAddress)
	}
	if *exportDirectory != "" {
		exporters = append(exporters, NewFileExporter(*exportDirectory))
		log.Print("Saveing traces to local directory: " + *exportDirectory)
	}
	if *jaegerServiceName != "" && *jaegerCollectorAddress != "" && *jaegerAgentAddress != "" {
		jaeger, err := NewJaegerExporter(*jaegerServiceName, *jaegerCollectorAddress, *jaegerAgentAddress)
		if err != nil {
			panic(err)
		}
		exporters = append(exporters, jaeger)
		log.Printf("Sending traces to Jaeger instance as %q: %s / %s", *jaegerServiceName, *jaegerCollectorAddress, *jaegerAgentAddress)
	}
	if len(exporters) == 0 {
		flag.Usage()
		log.Fatal("No exporters specified")
	}

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

		for _, exporter := range exporters {
			for _, trace := range traces {
				exporter.Export(trace)
				exporter.Flush()
			}
		}

	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	log.Println("Listing on " + *addr)
	http.ListenAndServe(*addr, nil)
}
