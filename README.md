# xfccprocessor

`xfccprocessor` is a custom OpenTelemetry Collector processor that expands values from Envoy XFCC (`x-forwarded-client-cert`) into Resource Attributes.

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
  xfccprocessor: {}
```

- `target_attribute`: Resource Attribute to write the extracted subject to (default: `tls.client.subject`)
- `overwrite`: whether to overwrite an existing `target_attribute` value (default: `false`)
- `include_certificates`: whether to expand heavy certificate values (`Cert` and `Chain`) (default: `false`)

## Attributes

The processor writes these attributes when the corresponding XFCC field is present:

| XFCC field | Resource Attribute |
| --- | --- |
| `Subject` | `target_attribute` (default: `tls.client.subject`) |
| `By` | `xfcc.by` |
| `Hash` | `xfcc.hash` |
| `URI` | `xfcc.uri` |
| `DNS` | `xfcc.dns` |

Certificate payloads are skipped by default. Enable `include_certificates` to also write:

| XFCC field | Resource Attribute |
| --- | --- |
| `Cert` | `tls.client.certificate` |
| `Chain` | `tls.client.certificate_chain` |

## Usage

```yaml
receivers:
  otlp:
    protocols:
      grpc:
      http:

processors:
  xfccprocessor:
    include_certificates: false

exporters:
  debug:

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [xfccprocessor]
      exporters: [debug]
    metrics:
      receivers: [otlp]
      processors: [xfccprocessor]
      exporters: [debug]
    logs:
      receivers: [otlp]
      processors: [xfccprocessor]
      exporters: [debug]
```

## Notes

This repository contains the processor implementation. To include it in a Collector distribution, register `NewFactory()` with `otelcol-builder` or an equivalent build setup.
