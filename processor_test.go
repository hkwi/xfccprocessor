package xfccprocessor

import (
	"testing"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestDefaultTargetAttributeUsesOTelTLSClientSubject(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	if cfg.TargetAttribute != "tls.client.subject" {
		t.Fatalf("TargetAttribute=%q, want %q", cfg.TargetAttribute, "tls.client.subject")
	}
	if cfg.IncludeCertificates {
		t.Fatal("IncludeCertificates=true, want false")
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

func TestProcessResourceExtractsLightXFCCAttributesByDefault(t *testing.T) {
	p := newProcessor(createDefaultConfig().(*Config))
	resource := pcommon.NewResource()
	resource.Attributes().PutStr(
		"http.request.header.x-forwarded-client-cert",
		`By=proxy;Hash=abc;Subject="CN=client.example";URI=spiffe://client;DNS=client.example;Cert="pem";Chain="chain"`,
	)

	p.processResource(resource)

	assertAttr(t, resource, "tls.client.subject", "CN=client.example")
	assertAttr(t, resource, "xfcc.by", "proxy")
	assertAttr(t, resource, "xfcc.hash", "abc")
	assertAttr(t, resource, "xfcc.uri", "spiffe://client")
	assertAttr(t, resource, "xfcc.dns", "client.example")
	assertMissingAttr(t, resource, "tls.client.certificate")
	assertMissingAttr(t, resource, "tls.client.certificate_chain")
}

func TestProcessResourceExtractsCertificatesWhenEnabled(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.IncludeCertificates = true
	p := newProcessor(cfg)
	resource := pcommon.NewResource()
	resource.Attributes().PutStr(
		"http.request.header.x-forwarded-client-cert",
		`By=proxy;Subject="CN=client.example";Cert="pem";Chain="chain"`,
	)

	p.processResource(resource)

	assertAttr(t, resource, "tls.client.certificate", "pem")
	assertAttr(t, resource, "tls.client.certificate_chain", "chain")
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

func assertAttr(t *testing.T, resource pcommon.Resource, key string, want string) {
	t.Helper()
	got, ok := resource.Attributes().Get(key)
	if !ok {
		t.Fatalf("missing %s", key)
	}
	if got.Str() != want {
		t.Fatalf("%s=%q, want %q", key, got.Str(), want)
	}
}

func assertMissingAttr(t *testing.T, resource pcommon.Resource, key string) {
	t.Helper()
	if _, ok := resource.Attributes().Get(key); ok {
		t.Fatalf("unexpected %s", key)
	}
}
