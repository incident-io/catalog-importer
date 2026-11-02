package config

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	parseConfig := func(pipelines string) *Config {
		cfg, err := parse([]byte(`{"sync_id": "something", "pipelines": ` + pipelines + `}`))
		Expect(err).NotTo(HaveOccurred())
		return cfg
	}

	const inlineSource = `[{"inline": {"entries": [{"schedule": "Primary on-call", "handle": "primary"}]}}]`
	const syncedTypeOutput = `{
		"type_name": "Schedule",
		"source": {"match": "$.schedule", "custom_aliases": ["$.handle"]}
	}`

	Describe("Validate", func() {
		It("accepts a pipeline with only synced type outputs", func() {
			cfg := parseConfig(`[{"sources": ` + inlineSource + `, "synced_type_outputs": [` + syncedTypeOutput + `]}]`)
			Expect(cfg.Validate()).To(Succeed())
			Expect(cfg.SyncedTypeOutputs()).To(HaveLen(1))
			Expect(cfg.SyncedTypeOutputs()[0].Source.CustomAliases).To(Equal([]string{"$.handle"}))
		})

		It("rejects a pipeline with no outputs of either kind", func() {
			cfg := parseConfig(`[{"sources": ` + inlineSource + `}]`)
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("must specify at least one output or synced type output")))
		})

		It("validates synced type outputs", func() {
			cfg := parseConfig(`[{"sources": ` + inlineSource + `, "synced_type_outputs": [{"type_name": "Schedule", "source": {"match": ""}}]}]`)
			Expect(cfg.Validate()).To(HaveOccurred())
		})
	})

	Describe("Filter", func() {
		It("keeps synced type outputs that match the targets", func() {
			cfg := parseConfig(`[
				{"sources": ` + inlineSource + `, "synced_type_outputs": [` + syncedTypeOutput + `]},
				{"sources": ` + inlineSource + `, "outputs": [{"name": "Team", "description": "Teams", "type_name": "Custom[\"Team\"]", "source": {"name": "$.name", "external_id": "$.id"}}]}
			]`)

			filtered := cfg.Filter([]string{"Schedule"})
			Expect(filtered.Pipelines).To(HaveLen(1))
			Expect(filtered.Pipelines[0].TypeNames()).To(Equal([]string{"Schedule"}))
		})
	})
})
