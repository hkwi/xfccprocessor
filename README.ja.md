# xfccsubjectprocessor

Envoy XFCC (`x-forwarded-client-cert`) から `subject` を抽出して、OpenTelemetry Resource Attribute に保存するカスタム Processor です。

OpenTelemetry Semantic Conventions で client certificate subject は `tls.client.subject` として定義されているため、デフォルトの保存先は `tls.client.subject` です。

## 対応フォーマット

- `text` 形式 (例: `By=...;Hash=...;Subject="CN=client";URI=...`)
- `json` 形式 (例: `[{"subject":"CN=client"}]` / 配列・ネストも探索)

入力元は OpenTelemetry の HTTP header attribute `http.request.header.x-forwarded-client-cert` 固定です。値は文字列と文字列配列の両方に対応します。

Envoy の仕様では text 形式の key は case-insensitive、`Subject` は double quote され、JSON 形式は object 配列です。この processor は実装差・中継差への耐性として、次の揺れも許容します。

- text 形式の single quote
- quote 内の `,` / `;` / `=` と escaped quote
- quote 外の escaped separator (`\,` / `\;` / `\=`)
- percent-encoded subject
- 複数 HTTP header 値の結合で JSON 配列が `[...] , [...]` になった値
- JSON field 名の大文字小文字違い、ネスト、subject 配列

## Config

```yaml
processors:
  xfccsubject: {}
```

- `target_attribute`: 抽出した subject を書き込む Resource Attribute (default: `tls.client.subject`)
- `overwrite`: 既存の `target_attribute` を上書きするか (default: `false`)

## 使い方（Collector 設定例）

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

## 注意

このリポジトリは Processor 実装本体です。Collector ディストリビューションに組み込むには、`otelcol-builder` 等で `NewFactory()` を登録してください。
