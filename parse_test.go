package xfccprocessor

import "testing"

func TestExtractSubjectFromText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		found bool
	}{
		{
			name:  "single text record",
			input: `By=spiffe://proxy/ns/default/sa/default;Hash=abc;Subject="CN=client.example";URI=spiffe://client`,
			want:  "CN=client.example",
			found: true,
		},
		{
			name:  "multi record takes first subject",
			input: `By=proxy-a;Subject="CN=first",By=proxy-b;Subject="CN=second"`,
			want:  "CN=first",
			found: true,
		},
		{
			name:  "case-insensitive key",
			input: `By=proxy;subject="CN=lowercase"`,
			want:  "CN=lowercase",
			found: true,
		},
		{
			name:  "quoted separators",
			input: `By=proxy;Subject="CN=client;OU=edge,DC=example";URI=spiffe://client`,
			want:  "CN=client;OU=edge,DC=example",
			found: true,
		},
		{
			name:  "quoted equals",
			input: `By=proxy;Subject="CN=client=one";URI=spiffe://client`,
			want:  "CN=client=one",
			found: true,
		},
		{
			name:  "single quoted subject",
			input: `By=proxy;Subject='CN=single-quoted';URI=spiffe://client`,
			want:  "CN=single-quoted",
			found: true,
		},
		{
			name:  "lenient escaped quotes",
			input: `By=proxy;Subject="CN=\"client\"";URI=spiffe://client`,
			want:  `CN="client"`,
			found: true,
		},
		{
			name:  "percent encoded subject",
			input: `By=proxy;Subject=CN%3Dencoded%3BOU%3Dedge;URI=spiffe://client`,
			want:  "CN=encoded;OU=edge",
			found: true,
		},
		{
			name:  "escaped separators outside quotes",
			input: `By=proxy;Subject=CN\=escaped\;OU\=edge\,DC\=example;URI=spiffe://client`,
			want:  "CN=escaped;OU=edge,DC=example",
			found: true,
		},
		{
			name:  "missing subject",
			input: `By=proxy;Hash=abc`,
			want:  "",
			found: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractSubject(tt.input)
			if ok != tt.found {
				t.Fatalf("found=%v, want=%v", ok, tt.found)
			}
			if got != tt.want {
				t.Fatalf("got=%q, want=%q", got, tt.want)
			}
		})
	}
}

func TestExtractSubjectFromJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		found bool
	}{
		{
			name:  "flat object",
			input: `{"subject":"CN=json-client"}`,
			want:  "CN=json-client",
			found: true,
		},
		{
			name:  "envoy array of objects",
			input: `[{"by":"proxy-a"},{"subject":"CN=array-client"}]`,
			want:  "CN=array-client",
			found: true,
		},
		{
			name:  "nested object",
			input: `{"xfcc":{"Subject":"CN=nested-client"}}`,
			want:  "CN=nested-client",
			found: true,
		},
		{
			name:  "joined json arrays",
			input: `[{"by":["proxy-a"]}],[{"subject":"CN=joined-json"}]`,
			want:  "CN=joined-json",
			found: true,
		},
		{
			name:  "subject array",
			input: `{"subject":["CN=array-subject"]}`,
			want:  "CN=array-subject",
			found: true,
		},
		{
			name:  "invalid json falls back and fails",
			input: `{invalid}`,
			want:  "",
			found: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractSubject(tt.input)
			if ok != tt.found {
				t.Fatalf("found=%v, want=%v", ok, tt.found)
			}
			if got != tt.want {
				t.Fatalf("got=%q, want=%q", got, tt.want)
			}
		})
	}
}

func TestExtractFieldsFromText(t *testing.T) {
	got, ok := ExtractFields(`By=proxy;Hash=abc;Subject="CN=client";URI=spiffe://client;DNS=client.example;Cert="pem";Chain="chain"`)
	if !ok {
		t.Fatal("missing fields")
	}

	want := map[string]string{
		"by":      "proxy",
		"hash":    "abc",
		"subject": "CN=client",
		"uri":     "spiffe://client",
		"dns":     "client.example",
		"cert":    "pem",
		"chain":   "chain",
	}

	for key, wantValue := range want {
		if got[key] != wantValue {
			t.Fatalf("%s=%q, want %q", key, got[key], wantValue)
		}
	}
}

func TestExtractFieldsFromJSON(t *testing.T) {
	got, ok := ExtractFields(`[{"by":"proxy"},{"hash":"abc","subject":"CN=client","uri":"spiffe://client","dns":["client.example"],"cert":"pem","chain":"chain"}]`)
	if !ok {
		t.Fatal("missing fields")
	}

	want := map[string]string{
		"by":      "proxy",
		"hash":    "abc",
		"subject": "CN=client",
		"uri":     "spiffe://client",
		"dns":     "client.example",
		"cert":    "pem",
		"chain":   "chain",
	}

	for key, wantValue := range want {
		if got[key] != wantValue {
			t.Fatalf("%s=%q, want %q", key, got[key], wantValue)
		}
	}
}
