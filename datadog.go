package main

type datadogSpan struct {
	ParentID uint64                 `json:"parent_id"`
	TraceID  uint64                 `json:"trace_id"`
	Type     string                 `json:"type"`
	Metrics  map[string]interface{} `json:"metrics"`
	Duration uint64                 `json:"duration"`
	SpanID   uint64                 `json:"span_id"`
	Name     string                 `json:"name"`
	Service  string                 `json:"service"`
	Resource string                 `json:"resource"`
	Meta     map[string]string      `json:"meta"`
	Error    int64                  `json:"error"`
	Start    uint64                 `json:"start"`
}
