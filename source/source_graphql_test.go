package source_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	kitlog "github.com/go-kit/kit/log"
	"github.com/incident-io/catalog-importer/v2/source"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SourceGraphQL", func() {
	var (
		ctx    context.Context
		logger kitlog.Logger
		server *httptest.Server
		body   string
	)

	BeforeEach(func() {
		ctx = context.Background()
		logger = kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stderr))
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, body)
		}))
	})

	AfterEach(func() {
		server.Close()
	})

	load := func() ([]*source.SourceEntry, error) {
		var s source.SourceGraphQL
		raw := fmt.Sprintf(
			`{"endpoint":%q,"query":"query{items}","result":"items"}`, server.URL)
		Expect(json.Unmarshal([]byte(raw), &s)).To(Succeed())

		return s.Load(ctx, logger, server.Client())
	}

	Describe("Load", func() {
		It("returns entries when the result is a list", func() {
			body = `{"data":{"items":[{"name":"service-a"}]}}`

			entries, err := load()
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(1))
			Expect(string(entries[0].Content)).To(MatchJSON(`[{"name":"service-a"}]`))
		})

		It("errors when the result is an object", func() {
			body = `{"data":{"items":{"name":"service-a"}}}`

			entries, err := load()
			Expect(err).To(MatchError(ContainSubstring("result is not a slice of values")))
			Expect(entries).To(BeNil())
		})

		It("errors instead of panicking when the result is null", func() {
			body = `{"data":{"items":null}}`

			entries, err := load()
			Expect(err).To(MatchError(ContainSubstring("result is not a slice of values")))
			Expect(entries).To(BeNil())
		})

		It("errors instead of panicking when the result path is missing", func() {
			body = `{"data":{"other":[]}}`

			entries, err := load()
			Expect(err).To(MatchError(ContainSubstring("result is not a slice of values")))
			Expect(entries).To(BeNil())
		})
	})
})
