package config

import (
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
})
