package xfccprocessor

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

type XFCCFields map[string]string

func ExtractSubject(xfcc string) (string, bool) {
	fields, ok := ExtractFields(xfcc)
	if !ok {
		return "", false
	}
	subject := fields["subject"]
	return subject, subject != ""
}

func ExtractFields(xfcc string) (XFCCFields, bool) {
	xfcc = strings.TrimSpace(xfcc)
	if xfcc == "" {
		return nil, false
	}

	if strings.HasPrefix(xfcc, "{") || strings.HasPrefix(xfcc, "[") {
		if fields, ok := extractFieldsFromJSON(xfcc); ok {
			return fields, true
		}
		if fields, ok := extractFieldsFromJoinedJSON(xfcc); ok {
			return fields, true
		}
	}

	if fields, ok := extractFieldsFromText(xfcc); ok {
		return fields, true
	}

	return nil, false
}

func extractFieldsFromJSON(raw string) (XFCCFields, bool) {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, false
	}
	fields := XFCCFields{}
	findFields(payload, fields)
	return fields, len(fields) > 0
}

func extractFieldsFromJoinedJSON(raw string) (XFCCFields, bool) {
	fields := XFCCFields{}
	for _, part := range splitTopLevel(raw, ',') {
		part = strings.TrimSpace(part)
		if part == "" || !(strings.HasPrefix(part, "{") || strings.HasPrefix(part, "[")) {
			continue
		}
		if partFields, ok := extractFieldsFromJSON(part); ok {
			mergeFirst(fields, partFields)
		}
	}
	return fields, len(fields) > 0
}

func findFields(v any, fields XFCCFields) {
	switch vv := v.(type) {
	case map[string]any:
		for k, value := range vv {
			if fieldKey, ok := normalizeFieldKey(k); ok {
				if _, exists := fields[fieldKey]; !exists {
					if s, ok := fieldString(value); ok && s != "" {
						fields[fieldKey] = s
					}
				}
			}
		}
		for _, value := range vv {
			findFields(value, fields)
		}
	case []any:
		for _, value := range vv {
			findFields(value, fields)
		}
	}
}

func fieldString(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, v != ""
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				return s, true
			}
		}
	}
	return "", false
}

func extractFieldsFromText(raw string) (XFCCFields, bool) {
	fields := XFCCFields{}
	entries := splitTopLevel(raw, ',')
	for _, entry := range entries {
		pairs := splitTopLevel(entry, ';')
		for _, pair := range pairs {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}

			kv := strings.SplitN(pair, "=", 2)
			if len(kv) != 2 {
				continue
			}

			fieldKey, ok := normalizeFieldKey(strings.TrimSpace(kv[0]))
			if !ok {
				continue
			}
			if _, exists := fields[fieldKey]; exists {
				continue
			}

			value := strings.TrimSpace(kv[1])
			if value == "" {
				continue
			}

			if unquoted, ok := unquote(value); ok {
				value = unquoted
			} else {
				value = unescapeLenient(value)
			}

			if decoded, ok := percentDecode(value); ok {
				value = decoded
			}

			if value != "" {
				fields[fieldKey] = value
			}
		}
	}
	return fields, len(fields) > 0
}

func normalizeFieldKey(key string) (string, bool) {
	switch strings.ToLower(key) {
	case "by":
		return "by", true
	case "hash":
		return "hash", true
	case "subject":
		return "subject", true
	case "uri":
		return "uri", true
	case "dns":
		return "dns", true
	case "cert":
		return "cert", true
	case "chain":
		return "chain", true
	}
	return "", false
}

func mergeFirst(dst, src XFCCFields) {
	for key, value := range src {
		if value == "" {
			continue
		}
		if _, exists := dst[key]; !exists {
			dst[key] = value
		}
	}
}

func unquote(value string) (string, bool) {
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		s, err := strconv.Unquote(value)
		if err == nil {
			return s, true
		}
		return unescapeLenient(value[1 : len(value)-1]), true
	}

	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return unescapeLenient(value[1 : len(value)-1]), true
	}
	return value, false
}

func unescapeLenient(value string) string {
	value = strings.ReplaceAll(value, `\"`, `"`)
	value = strings.ReplaceAll(value, `\'`, `'`)
	value = strings.ReplaceAll(value, `\,`, `,`)
	value = strings.ReplaceAll(value, `\;`, `;`)
	value = strings.ReplaceAll(value, `\=`, `=`)
	value = strings.ReplaceAll(value, `\\`, `\`)
	return value
}

func percentDecode(value string) (string, bool) {
	if !strings.Contains(value, "%") {
		return "", false
	}
	decoded, err := url.QueryUnescape(value)
	if err != nil || decoded == "" || !utf8.ValidString(decoded) {
		return "", false
	}
	return decoded, true
}

func splitTopLevel(input string, separator rune) []string {
	var parts []string
	var current strings.Builder
	var quote rune
	escape := false
	bracketDepth := 0
	braceDepth := 0

	for _, r := range input {
		switch {
		case escape:
			current.WriteRune(r)
			escape = false
		case r == '\\':
			current.WriteRune(r)
			escape = true
		case quote != 0:
			current.WriteRune(r)
			if r == quote {
				quote = 0
			}
		case r == '"' || r == '\'':
			current.WriteRune(r)
			quote = r
		case r == '[':
			current.WriteRune(r)
			bracketDepth++
		case r == ']':
			current.WriteRune(r)
			if bracketDepth > 0 {
				bracketDepth--
			}
		case r == '{':
			current.WriteRune(r)
			braceDepth++
		case r == '}':
			current.WriteRune(r)
			if braceDepth > 0 {
				braceDepth--
			}
		case r == separator && bracketDepth == 0 && braceDepth == 0:
			parts = append(parts, strings.TrimSpace(current.String()))
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}

	parts = append(parts, strings.TrimSpace(current.String()))
	return parts
}
