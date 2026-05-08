package xfccsubjectprocessor

import (
	"testing"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestDefaultTargetAttributeUsesOTelTLSClientSubject(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	if cfg.TargetAttribute != "tls.client.subject" {
		t.Fatalf("TargetAttribute=%q, want %q", cfg.TargetAttribute, "tls.client.subject")
	}
}

func TestProcessResourceExtractsSubjectFromStringAttribute(t *testing.T) {
	p := newProcessor(createDefaultConfig().(*Config))
	resource := pcommon.NewResource()
	resource.Attributes().PutStr(
		"http.request.header.x-forwarded-client-cert",
		`By=proxy;Subject="CN=client.example";URI=spiffe://client`,
	)

	p.processResource(resource)

	got, ok := resource.Attributes().Get("tls.client.subject")
	if !ok {
		t.Fatal("missing tls.client.subject")
	}
	if got.Str() != "CN=client.example" {
		t.Fatalf("got %q, want %q", got.Str(), "CN=client.example")
	}
}

func TestProcessResourceExtractsSubjectFromStringSliceAttribute(t *testing.T) {
	p := newProcessor(createDefaultConfig().(*Config))
	resource := pcommon.NewResource()
	values := resource.Attributes().PutEmptySlice("http.request.header.x-forwarded-client-cert")
	values.AppendEmpty().SetStr(`By=proxy-a;Hash=abc`)
	values.AppendEmpty().SetStr(`[{"subject":"CN=json-client"}]`)

	p.processResource(resource)

	got, ok := resource.Attributes().Get("tls.client.subject")
	if !ok {
		t.Fatal("missing tls.client.subject")
	}
	if got.Str() != "CN=json-client" {
		t.Fatalf("got %q, want %q", got.Str(), "CN=json-client")
	}
}

func TestProcessResourceKeepsExistingTargetByDefault(t *testing.T) {
	p := newProcessor(createDefaultConfig().(*Config))
	resource := pcommon.NewResource()
	resource.Attributes().PutStr("tls.client.subject", "CN=existing")
	resource.Attributes().PutStr(
		"http.request.header.x-forwarded-client-cert",
		`By=proxy;Subject="CN=new";URI=spiffe://client`,
	)

	p.processResource(resource)

	got, _ := resource.Attributes().Get("tls.client.subject")
	if got.Str() != "CN=existing" {
		t.Fatalf("got %q, want %q", got.Str(), "CN=existing")
	}
}
