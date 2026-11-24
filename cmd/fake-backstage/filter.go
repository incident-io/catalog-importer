package main

import (
	"fmt"
	"strings"
)

// filter is one `filter` query parameter, following the Backstage catalog API rules:
//
//   - conditions are comma-separated, as key=value, or key alone to mean "has this key"
//   - a key repeated within one filter matches any of its values
//   - different keys within one filter must all match
//   - keys and values are compared case-insensitively
//
// Several `filter` parameters match an entity if any one of them does.
type filter map[string][]string

func parseFilter(s string) filter {
	f := filter{}
	for _, statement := range strings.Split(s, ",") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		key, value, hasValue := strings.Cut(statement, "=")
		key = strings.ToLower(strings.TrimSpace(key))
		if !hasValue {
			f[key] = append(f[key], "")
			continue
		}
		f[key] = append(f[key], strings.ToLower(strings.TrimSpace(value)))
	}
	return f
}

func (f filter) matches(fields map[string][]string) bool {
	for key, want := range f {
		have, ok := fields[key]
		if !ok {
			return false
		}
		if !anyMatch(want, have) {
			return false
		}
	}
	return true
}

func anyMatch(want, have []string) bool {
	for _, w := range want {
		if w == "" {
			return true // key-only condition: the key exists
		}
		for _, h := range have {
			if w == h {
				return true
			}
		}
	}
	return false
}

// flatten turns an entity into the dotted keys Backstage filters on, such as
// metadata.name or metadata.annotations.example.com/on-call-team. Array elements share
// their parent's key, and relations are indexed as relations.<type> = <target ref>.
func flatten(e entity) map[string][]string {
	fields := map[string][]string{}
	var walk func(prefix string, v any)
	walk = func(prefix string, v any) {
		switch v := v.(type) {
		case map[string]any:
			for k, child := range v {
				key := strings.ToLower(k)
				if prefix != "" {
					key = prefix + "." + key
				}
				walk(key, child)
			}
		case []any:
			for _, child := range v {
				walk(prefix, child)
			}
		case nil:
		default:
			fields[prefix] = append(fields[prefix], strings.ToLower(fmt.Sprint(v)))
		}
	}

	for k, v := range e {
		if k == "relations" {
			continue
		}
		walk(strings.ToLower(k), v)
	}

	if relations, ok := e["relations"].([]any); ok {
		for _, r := range relations {
			rel, _ := r.(map[string]any)
			key := "relations." + strings.ToLower(fmt.Sprint(rel["type"]))
			fields[key] = append(fields[key], strings.ToLower(fmt.Sprint(rel["targetRef"])))
		}
	}

	return fields
}
