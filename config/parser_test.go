package config

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parse", func() {
	It("parses valid config", func() {
		cfg, err := parse([]byte(`
{
	"sync_id": "something",
	"pipelines": [
		{
			"sources": [
				{
					"inline": {
						"entries": [{
							"external_id": "entry-external-id",
							"name": "entry-name",
							"description": "entry-description",
						}]
					}
				}
			],
			"outputs": []
		}
	]
}`))
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Pipelines).To(HaveLen(1))
		Expect(cfg.Pipelines[0].Sources).To(HaveLen(1))
	})

	It("parses valid config", func() {
		_, err := parse([]byte(`
{
	"sync_id": "something",
	"pipelines": [
		{
			"invalid_key": [
				{
					"inline": {
						"entries": [{
							"external_id": "entry-external-id",
							"name": "entry-name",
							"description": "entry-description",
						}]
					}
				}
			]
		}
	]
}`))
		Expect(err).To(MatchError(ContainSubstring("unknown field \"invalid_key\"")))
	})

	It("resolves Jsonnet imports relative to the config file", func() {
		dir := GinkgoT().TempDir()
		Expect(os.MkdirAll(filepath.Join(dir, "pipelines"), 0o755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "pipelines", "inline.jsonnet"), []byte(`{
			sources: [{ inline: { entries: [] } }],
			outputs: [],
		}`), 0o644)).To(Succeed())

		filename := filepath.Join(dir, "importer.jsonnet")
		data := []byte(`{ sync_id: 'something', pipelines: [import 'pipelines/inline.jsonnet'] }`)

		cfg, err := Parse(filename, data)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Pipelines).To(HaveLen(1))
	})
})
