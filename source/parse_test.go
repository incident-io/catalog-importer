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

		When("std.thisFile", func() {
			BeforeEach(func() {
				input = `
{
	name: std.thisFile,
}
`
			})

			It("returns filename", func() {
				Expect(entries).To(Equal([]source.Entry{
					{
						"name": "file.thing",
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
				Expect(entries).To(Equal([]source.Entry{
					{
						"id":          "P123",
						"name":        "My name is",
						"description": "What",
					},
					{
						"id":          "P124",
						"name":        "My name is",
						"description": "Who",
					},
					{
						"id":          "P125",
						"name":        "My name is",
						"description": "Slim Shady",
					},
				}))
			})
		})
	})
})
