package xfccsubjectprocessor

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

func ExtractSubject(xfcc string) (string, bool) {
	xfcc = strings.TrimSpace(xfcc)
	if xfcc == "" {
		return "", false
	}

	if strings.HasPrefix(xfcc, "{") || strings.HasPrefix(xfcc, "[") {
		if subject, ok := extractFromJSON(xfcc); ok {
			return subject, true
		}
		if subject, ok := extractFromJoinedJSON(xfcc); ok {
			return subject, true
		}
	}

	if subject, ok := extractFromText(xfcc); ok {
		return subject, true
	}

	return "", false
}

func extractFromJSON(raw string) (string, bool) {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", false
	}
	return findSubject(payload)
}

func extractFromJoinedJSON(raw string) (string, bool) {
	for _, part := range splitTopLevel(raw, ',') {
		part = strings.TrimSpace(part)
		if part == "" || !(strings.HasPrefix(part, "{") || strings.HasPrefix(part, "[")) {
			continue
		}
		if subject, ok := extractFromJSON(part); ok {
			return subject, true
		}
	}
	return "", false
}

func findSubject(v any) (string, bool) {
	switch vv := v.(type) {
	case map[string]any:
		for k, value := range vv {
			if strings.EqualFold(k, "subject") {
				if s, ok := subjectString(value); ok && s != "" {
					return s, true
				}
			}
		}
		for _, value := range vv {
			if s, ok := findSubject(value); ok {
				return s, true
			}
		}
	case []any:
		for _, value := range vv {
			if s, ok := findSubject(value); ok {
				return s, true
			}
		}
	}
	return "", false
}

func subjectString(value any) (string, bool) {
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

func extractFromText(raw string) (string, bool) {
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

			if !strings.EqualFold(strings.TrimSpace(kv[0]), "subject") {
				continue
			}

			value := strings.TrimSpace(kv[1])
			if value == "" {
				return "", false
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
				return value, true
			}
		}
	}
	return "", false
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
