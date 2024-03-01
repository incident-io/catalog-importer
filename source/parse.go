package source

import (
	"bytes"
	"encoding/csv"
	"strings"
	"unicode"

	"github.com/ghodss/yaml"
	"github.com/google/go-jsonnet"
)

// Entry is a single sourced entry.  It's just a basic map, but makes it much clearer when
// building lists of this type, as the type syntax can get a bit messy.
type Entry map[string]any

// Parse attempts to extract entries from content that is either Jsonnet, JSON or YAML.
//
// It also supports multidoc YAML, and will either return the root object itself if that
// root is a map[string]any, or if the root is an array, will try returning the contents
// of said array that are map[string]any's.
func Parse(filename string, data []byte) []Entry {
	// Try Jsonnet first, which will also cover JSON.
	jsonString, err := jsonnet.MakeVM().EvaluateSnippet(filename, string(data))
	if err == nil {
		data = []byte(jsonString)
	}

	// What we have now is either JSON or YAML.
	var (
		entries   = []Entry{}
		docChunks = bytes.Split(data, []byte("\n---"))
	)

	for _, chunk := range docChunks {
		{
			var entry map[string]any
			if err := yaml.Unmarshal(chunk, &entry); err != nil {
				goto tryArray
			}
			entries = append(entries, entry)
			continue
		}

	tryArray:
		{
			var listOfEntries []Entry
			if err := yaml.Unmarshal(chunk, &listOfEntries); err != nil {
				continue
			}

			entries = append(entries, listOfEntries...)
		}
	}

	// If we find nothing, we'll attempt CSV as a hail-mary.
	if len(entries) == 0 {
		records, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
		if err != nil {
			return entries
		}

		// We can only use CSVs that provide a header row. And if there only exists headers,
		// we should return no entries.
		if len(records) <= 1 {
			return entries
		}

		headers, rows := records[0], records[1:]
		headerIndexes := map[int]string{}
		for idx, header := range headers {
			headerStripped := strings.TrimFunc(header, func(r rune) bool {
				return !unicode.IsGraphic(r)
			})

			headerIndexes[idx] = headerStripped
		}

		for _, row := range rows {
			entry := Entry{}
			for idx, column := range row {
				entry[headerIndexes[idx]] = column
			}

			entries = append(entries, entry)
		}
	}

	return entries
}
