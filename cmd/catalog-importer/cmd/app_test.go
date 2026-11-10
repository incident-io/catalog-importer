package cmd

import (
	"bytes"
	"encoding/json"
	"strings"

	kitlog "github.com/go-kit/kit/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("progress output with --log-format json", func() {
	var buf *bytes.Buffer

	BeforeEach(func() {
		buf = &bytes.Buffer{}
		outputLogger = kitlog.NewJSONLogger(buf)
		DeferCleanup(func() {
			outputLogger = nil
			*quiet = false
		})
	})

	entries := func() []map[string]any {
		result := []map[string]any{}
		for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
			if line == "" {
				continue
			}
			entry := map[string]any{}
			Expect(json.Unmarshal([]byte(line), &entry)).To(Succeed(), line)
			result = append(result, entry)
		}

		return result
	}

	Describe("OUT", func() {
		It("writes one entry per call, without the terminal layout", func() {
			OUT("\n  ✔ %s (found %d entries)", "backstage", 3)

			Expect(entries()).To(Equal([]map[string]any{
				{"level": "info", "msg": "✔ backstage (found 3 entries)"},
			}))
		})

		It("writes nothing for blank output", func() {
			OUT("\n")

			Expect(buf.String()).To(BeEmpty())
		})

		It("writes nothing with --quiet", func() {
			*quiet = true
			OUT("✔ Loaded config")

			Expect(buf.String()).To(BeEmpty())
		})
	})

	Describe("BANNER", func() {
		It("writes the message without the banner", func() {
			BANNER("Running %s", "sync")

			Expect(entries()).To(Equal([]map[string]any{
				{"level": "info", "msg": "Running sync"},
			}))
		})
	})

	Describe("DIFF", func() {
		It("writes the whole diff as one entry", func() {
			DIFF("    ", map[string]any{"name": "Old"}, map[string]any{"name": "New"})

			result := entries()
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(HaveKeyWithValue("level", "info"))
			Expect(result[0]).To(HaveKeyWithValue("msg", "diff"))
			Expect(result[0]["diff"]).To(And(ContainSubstring(`- 	"name": string("Old")`), ContainSubstring(`+ 	"name": string("New")`)))
		})

		It("writes nothing when there's no difference", func() {
			DIFF("    ", map[string]any{"name": "Same"}, map[string]any{"name": "Same"})

			Expect(buf.String()).To(BeEmpty())
		})
	})
})
