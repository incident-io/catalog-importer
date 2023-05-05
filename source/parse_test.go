package source_test

import (
	"github.com/incident-io/catalog-importer/source"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parse", func() {
	var (
		input   string
		entries []source.Entry
	)

	JustBeforeEach(func() {
		entries = source.Parse("file.thing", []byte(input))
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
				Expect(entries).To(Equal([]source.Entry{
					{
						"key": "value",
						"nested": map[any]any{
							"another_key": "another_value",
						},
					},
				}))
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
				Expect(entries).To(Equal([]source.Entry{
					{
						"key": "value",
					},
					{
						"another_key": "another_value",
					},
				}))
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
				Expect(entries).To(Equal([]source.Entry{
					{
						"key": "value",
						"nested": map[any]any{
							"another_key": "another_value",
						},
					},
				}))
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
				Expect(entries).To(Equal([]source.Entry{
					{
						"key": "value",
					},
					{
						"another_key": "another_value",
					},
				}))
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
				Expect(entries).To(Equal([]source.Entry{
					{
						"key": "value",
						"nested": map[any]any{
							"another_key": "another_value",
						},
					},
				}))
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
				Expect(entries).To(Equal([]source.Entry{
					{
						"key": "value",
						"nested": map[any]any{
							"another_key": "another_value",
						},
					},
					{
						"we": "hate yaml",
					},
				}))
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
				Expect(entries).To(Equal([]source.Entry{
					{
						"key": "value",
					},
					{
						"another_key": "another_value",
					},
				}))
			})
		})
	})
})
