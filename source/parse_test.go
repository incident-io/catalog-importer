package source_test

import (
	"github.com/incident-io/catalog-importer/source"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("Parse", func() {
	var (
		input   string
		entries []source.Entry
		err     error
	)

	JustBeforeEach(func() {
		entries, err = source.Parse("README.md", []byte(input))
		Expect(err).ToNot(HaveOccurred())
	})

	When("Jsonnet", func() {
		When("object", func() {
			BeforeEach(func() {
				input = `
{
	key: "value",
	hidden:: false,
	nested: {
		another_key: "another_value",
	},
}
`
			})

			It("returns the object", func() {
				Expect(entries).To(ConsistOf(MatchKeys(
					IgnoreExtras, Keys{
						"_local.filename": Equal("README.md"),
						"_local.path":     HaveSuffix("README.md"),
						"key":             Equal("value"),
						"nested": MatchKeys(IgnoreExtras, Keys{
							"another_key": Equal("another_value"),
						}),
					},
				)))
			})
		})

		When("std.thisFile", func() {
			BeforeEach(func() {
				input = `
{
	name: std.thisFile,
}
`
			})

			It("returns filename", func() {
				Expect(entries).To(ConsistOf(MatchKeys(IgnoreExtras, Keys{
					"_local.filename": Equal("README.md"),
					"_local.path":     HaveSuffix("README.md"),
					"name":            Equal("README.md"),
				})))
			})
		})

		When("array", func() {
			BeforeEach(func() {
				input = `
[
	{
		key: "value",
	},
	{
		another_key: "another_value",
	}
]
`
			})

			It("returns all objects", func() {
				Expect(entries).To(ConsistOf(
					MatchKeys(IgnoreExtras, Keys{
						"_local.filename": Equal("README.md"),
						"_local.path":     HaveSuffix("README.md"),
						"key":             Equal("value"),
					}),
					MatchKeys(IgnoreExtras, Keys{
						"_local.filename": Equal("README.md"),
						"_local.path":     HaveSuffix("README.md"),
						"another_key":     Equal("another_value"),
					}),
				))
			})
		})
	})

	When("JSON", func() {
		When("object", func() {
			BeforeEach(func() {
				input = `
{
	"key": "value",
	"nested": {
		"another_key": "another_value",
	}
}
`
			})

			It("returns the object", func() {
				Expect(entries).To(ConsistOf(MatchKeys(
					IgnoreExtras, Keys{
						"_local.filename": Equal("README.md"),
						"_local.path":     HaveSuffix("README.md"),
						"key":             Equal("value"),
						"nested": MatchKeys(IgnoreExtras, Keys{
							"another_key": Equal("another_value"),
						}),
					},
				)))
			})
		})

		When("array", func() {
			BeforeEach(func() {
				input = `
[
	{
		"key": "value",
	},
	{
		"another_key": "another_value",
	}
]
`
			})

			It("returns all objects", func() {
				Expect(entries).To(ConsistOf(
					MatchKeys(IgnoreExtras, Keys{
						"_local.filename": Equal("README.md"),
						"_local.path":     HaveSuffix("README.md"),
						"key":             Equal("value"),
					}),
					MatchKeys(IgnoreExtras, Keys{
						"_local.filename": Equal("README.md"),
						"_local.path":     HaveSuffix("README.md"),
						"another_key":     Equal("another_value"),
					}),
				))
			})
		})
	})

	When("YAML", func() {
		When("object", func() {
			BeforeEach(func() {
				input = `
key: value
nested:
  another_key: another_value
`
			})

			It("returns the object", func() {
				Expect(entries).To(ConsistOf(MatchKeys(
					IgnoreExtras, Keys{
						"_local.filename": Equal("README.md"),
						"_local.path":     HaveSuffix("README.md"),
						"key":             Equal("value"),
						"nested": MatchKeys(IgnoreExtras, Keys{
							"another_key": Equal("another_value"),
						}),
					},
				)))
			})
		})

		When("multidoc", func() {
			BeforeEach(func() {
				input = `
key: value
nested:
  another_key: another_value
---
we: hate yaml
`
			})

			It("returns the object", func() {
				Expect(entries).To(ConsistOf(
					MatchKeys(
						IgnoreExtras, Keys{
							"_local.filename": Equal("README.md"),
							"_local.path":     HaveSuffix("README.md"),
							"key":             Equal("value"),
							"nested": MatchKeys(IgnoreExtras, Keys{
								"another_key": Equal("another_value"),
							}),
						},
					),
					MatchKeys(
						IgnoreExtras, Keys{
							"_local.filename": Equal("README.md"),
							"_local.path":     HaveSuffix("README.md"),
							"we":              Equal("hate yaml"),
						}),
				))
			})
		})

		When("array", func() {
			BeforeEach(func() {
				input = `
- key: "value"
- another_key: "another_value"
`
			})

			It("returns all objects", func() {
				Expect(entries).To(ConsistOf(
					MatchKeys(IgnoreExtras, Keys{
						"_local.filename": Equal("README.md"),
						"_local.path":     HaveSuffix("README.md"),
						"key":             Equal("value"),
					}),
					MatchKeys(IgnoreExtras, Keys{
						"_local.filename": Equal("README.md"),
						"_local.path":     HaveSuffix("README.md"),
						"another_key":     Equal("another_value"),
					}),
				))
			})
		})
	})

	When("CSV", func() {
		When("headers", func() {
			BeforeEach(func() {
				input = `
id,name,description
P123,My name is,What
P124,My name is,Who
P125,My name is,Slim Shady
`
			})

			It("returns all parsed entries", func() {
				Expect(entries).To(ConsistOf(
					MatchKeys(IgnoreExtras, Keys{
						"_local.filename": Equal("README.md"),
						"_local.path":     HaveSuffix("README.md"),
						"id":              Equal("P123"),
						"name":            Equal("My name is"),
						"description":     Equal("What"),
					}),
					MatchKeys(IgnoreExtras, Keys{
						"_local.filename": Equal("README.md"),
						"_local.path":     HaveSuffix("README.md"),
						"id":              Equal("P124"),
						"name":            Equal("My name is"),
						"description":     Equal("Who"),
					}),
					MatchKeys(IgnoreExtras, Keys{
						"_local.filename": Equal("README.md"),
						"_local.path":     HaveSuffix("README.md"),
						"id":              Equal("P125"),
						"name":            Equal("My name is"),
						"description":     Equal("Slim Shady"),
					}),
				))
			})
		})
	})
})
