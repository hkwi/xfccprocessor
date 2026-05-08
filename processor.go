package xfccprocessor

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type xfccProcessor struct {
	targetAttr          string
	overwrite           bool
	includeCertificates bool
}

const sourceAttribute = "http.request.header.x-forwarded-client-cert"

const (
	attributeXFCCBy               = "xfcc.by"
	attributeXFCCHash             = "xfcc.hash"
	attributeXFCCURI              = "xfcc.uri"
	attributeXFCCDNS              = "xfcc.dns"
	attributeTLSClientCertificate = "tls.client.certificate"
	attributeTLSClientCertChain   = "tls.client.certificate_chain"
)

func newProcessor(cfg *Config) *xfccProcessor {
	return &xfccProcessor{
		targetAttr:          cfg.TargetAttribute,
		overwrite:           cfg.Overwrite,
		includeCertificates: cfg.IncludeCertificates,
	}
}

func (p *xfccProcessor) processTraces(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		p.processResource(rss.At(i).Resource())
	}
	return td, nil
}

func (p *xfccProcessor) processMetrics(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		p.processResource(rms.At(i).Resource())
	}
	return md, nil
}

func (p *xfccProcessor) processLogs(_ context.Context, ld plog.Logs) (plog.Logs, error) {
	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		p.processResource(rls.At(i).Resource())
	}
	return ld, nil
}

func (p *xfccProcessor) processResource(resource pcommon.Resource) {
	attrs := resource.Attributes()

	xfccValue, ok := attrs.Get(sourceAttribute)
	if !ok {
		return
	}

	fields, ok := extractFieldsFromAttribute(xfccValue)
	if !ok {
		return
	}

	p.putString(attrs, p.targetAttr, fields["subject"])
	p.putString(attrs, attributeXFCCBy, fields["by"])
	p.putString(attrs, attributeXFCCHash, fields["hash"])
	p.putString(attrs, attributeXFCCURI, fields["uri"])
	p.putString(attrs, attributeXFCCDNS, fields["dns"])
	if p.includeCertificates {
		p.putString(attrs, attributeTLSClientCertificate, fields["cert"])
		p.putString(attrs, attributeTLSClientCertChain, fields["chain"])
	}
}

func extractSubjectFromAttribute(value pcommon.Value) (string, bool) {
	fields, ok := extractFieldsFromAttribute(value)
	if !ok {
		return "", false
	}
	subject := fields["subject"]
	return subject, subject != ""
}

func extractFieldsFromAttribute(value pcommon.Value) (XFCCFields, bool) {
	fields := XFCCFields{}
	switch value.Type() {
	case pcommon.ValueTypeStr:
		return ExtractFields(value.Str())
	case pcommon.ValueTypeSlice:
		slice := value.Slice()
		for i := 0; i < slice.Len(); i++ {
			item := slice.At(i)
			if item.Type() != pcommon.ValueTypeStr {
				continue
			}
			if itemFields, ok := ExtractFields(item.Str()); ok {
				mergeFirst(fields, itemFields)
			}
		}
	}
	return fields, len(fields) > 0
}

func (p *xfccProcessor) putString(attrs pcommon.Map, key string, value string) {
	if value == "" {
		return
	}
	if !p.overwrite {
		if _, exists := attrs.Get(key); exists {
			return
		}
	}
	attrs.PutStr(key, value)
}
