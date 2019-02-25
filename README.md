Datadog Trace Proxy
===

Listens for Datadog APM traces in msgpack format and does one or more of:

* saves them in to a directory in JSON format
* converts and forwards them on to an Appdash instance
* converts and forwards them on to a Jaeger instance

## Running:

```
$ datadog-trace-proxy \
    -jaeger.service my-service \
    -jaeger.collector http://localhost:14268/api/traces \
    -jaeger.agent localhost:6831
```

Configure any Datadog traced services to use the address of the trace proxy.

## Configuring a Ruby/Rails application

Assuming you have `datadog-trace-proxy` listening on the same host on port 12345 you can specify the hostname and port of the Datadog agent to use:

```ruby
Datadog.configure do |c|
    c.tracer(
        enabled: true,
        env: Rails.env,
        hostname: "localhost",
        port: 12345,
    )
    c.use(:rails, distributed_tracing: true)
    c.use(:http, distributed_tracing: true)
end
```

## Notes:

The Jaeger exporting works pretty well.
The span name and attribute mapping is a bit sparse and is only really coded for HTTP spans.

This doesn't support receiving Datadog traces in JSON format, just msgpack.
I'd only recently noticed it was an option and haven't looked in to it.

The Datadog trace format was worked out via documentation and playing with what was received over HTTP.
It may be missing portions, or have incorrectly formatted fields.
It works well enough.

This doesn't have any tests, and could use some.

This isn't supported by or have anything official to do with Datadog.
