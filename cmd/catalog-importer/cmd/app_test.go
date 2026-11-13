package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
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
			// go-cmp randomly writes non-breaking spaces into its diffs, so they can't be
			// matched exactly.
			Expect(result[0]["diff"]).To(And(
				MatchRegexp(`(?m)^-[\s\x{a0}]+"name": string\("Old"\)`),
				MatchRegexp(`(?m)^\+[\s\x{a0}]+"name": string\("New"\)`),
			))
		})

		It("writes nothing when there's no difference", func() {
			DIFF("    ", map[string]any{"name": "Same"}, map[string]any{"name": "Same"})

			Expect(buf.String()).To(BeEmpty())
		})

		It("writes nothing with --quiet", func() {
			*quiet = true
			DIFF("    ", map[string]any{"name": "Old"}, map[string]any{"name": "New"})

			Expect(buf.String()).To(BeEmpty())
		})
	})

	Describe("Fatal", func() {
		var exitCode int

		BeforeEach(func() {
			exitCode = -1
			exit = func(code int) { exitCode = code }
			DeferCleanup(func() { exit = os.Exit })
		})

		It("writes the error as an error entry, and exits 1", func() {
			Fatal(errors.New("listing entries: GET /v2/catalog_entries?catalog_type_id=Custom%5B%22Team%22%5D: 500"))

			Expect(entries()).To(Equal([]map[string]any{
				{"level": "error", "msg": `listing entries: GET /v2/catalog_entries?catalog_type_id=Custom%5B%22Team%22%5D: 500`},
			}))
			Expect(exitCode).To(Equal(1))
		})
	})
})

var _ = Describe("Fatal without --log-format json", func() {
	It("writes the error in kingpin's format, and exits 1", func() {
		var buf bytes.Buffer
		exitCode := -1
		app.ErrorWriter(&buf)
		exit = func(code int) { exitCode = code }
		DeferCleanup(func() {
			app.ErrorWriter(os.Stderr)
			exit = os.Exit
		})

		Fatal(errors.New("loading config: open 100%.jsonnet: no such file or directory"))

		Expect(buf.String()).To(Equal("catalog-importer: error: loading config: open 100%.jsonnet: no such file or directory\n"))
		Expect(exitCode).To(Equal(1))
	})
})
