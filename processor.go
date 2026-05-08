package xfccsubjectprocessor

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type xfccProcessor struct {
	targetAttr string
	overwrite  bool
}

const sourceAttribute = "http.request.header.x-forwarded-client-cert"

func newProcessor(cfg *Config) *xfccProcessor {
	return &xfccProcessor{
		targetAttr: cfg.TargetAttribute,
		overwrite:  cfg.Overwrite,
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

	if !p.overwrite {
		if _, exists := attrs.Get(p.targetAttr); exists {
			return
		}
	}

	xfccValue, ok := attrs.Get(sourceAttribute)
	if !ok {
		return
	}

	subject, ok := extractSubjectFromAttribute(xfccValue)
	if !ok || subject == "" {
		return
	}

	attrs.PutStr(p.targetAttr, subject)
}

func extractSubjectFromAttribute(value pcommon.Value) (string, bool) {
	switch value.Type() {
	case pcommon.ValueTypeStr:
		return ExtractSubject(value.Str())
	case pcommon.ValueTypeSlice:
		slice := value.Slice()
		for i := 0; i < slice.Len(); i++ {
			item := slice.At(i)
			if item.Type() != pcommon.ValueTypeStr {
				continue
			}
			if subject, ok := ExtractSubject(item.Str()); ok {
				return subject, true
			}
		}
	}
	return "", false
}
