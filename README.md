# xfccsubjectprocessor

`xfccsubjectprocessor` is a custom OpenTelemetry Collector processor that extracts the `subject` value from Envoy XFCC (`x-forwarded-client-cert`) and stores it as a Resource Attribute.

Because OpenTelemetry Semantic Conventions define the client certificate subject as `tls.client.subject`, the default output attribute is `tls.client.subject`.

## Supported Formats

- `text` format (example: `By=...;Hash=...;Subject="CN=client";URI=...`)
- `json` format (example: `[{"subject":"CN=client"}]`; arrays and nested values are also searched)

The input attribute is fixed to the OpenTelemetry HTTP header attribute `http.request.header.x-forwarded-client-cert`. Both string and string-array values are supported.

In the Envoy specification, text-format keys are case-insensitive, `Subject` is double-quoted, and the JSON format is an array of objects. To tolerate implementation and proxy differences, this processor also accepts:

- single quotes in text format
- `,`, `;`, `=`, and escaped quotes inside quoted values
- escaped separators outside quoted values (`\,` / `\;` / `\=`)
- percent-encoded subject
- joined JSON arrays such as `[...] , [...]`, which can appear when multiple HTTP header values are combined
- case variations in JSON field names, nested values, and subject arrays

## Config

```yaml
processors:
  xfccsubject: {}
```

- `target_attribute`: Resource Attribute to write the extracted subject to (default: `tls.client.subject`)
- `overwrite`: whether to overwrite an existing `target_attribute` value (default: `false`)

## Usage

```yaml
receivers:
  otlp:
    protocols:
      grpc:
      http:

processors:
  xfccsubject: {}

exporters:
  debug:

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [xfccsubject]
      exporters: [debug]
    metrics:
      receivers: [otlp]
      processors: [xfccsubject]
      exporters: [debug]
    logs:
      receivers: [otlp]
      processors: [xfccsubject]
      exporters: [debug]
```

## Notes

This repository contains the processor implementation. To include it in a Collector distribution, register `NewFactory()` with `otelcol-builder` or an equivalent build setup.
