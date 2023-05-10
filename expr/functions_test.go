package expr

import (
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Functions", func() {
	var (
		src string
		prg cel.Program
	)

	JustBeforeEach(func() {
		env, err := cel.NewEnv(Stdlib())
		Expect(err).NotTo(HaveOccurred())

		ast, issues := env.Parse(src)
		Expect(issues.Err()).NotTo(HaveOccurred())

		prg, err = env.Program(ast)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("First", func() {
		When("the input is a valid list", func() {
			BeforeEach(func() {
				src = "first(['chinchilla', 'capybara', 'corgi'])"
			})

			It("returns the first item in an array", func() {
				out, _, err := prg.Eval(map[string]any{})
				Expect(err).NotTo(HaveOccurred())

				Expect(out).To(Equal(types.String("chinchilla")))
			})
		})

		When("the input is not a valid list", func() {
			BeforeEach(func() {
				src = "first('corgi')"
			})

			It("returns the first item in an array", func() {
				_, _, err := prg.Eval(map[string]any{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("no such overload: first(string)"))
			})
		})
	})

	Describe("Coalesce", func() {
		When("the input is a list of objects", func() {
			BeforeEach(func() {
				src = "coalesce([{'key': 'value'}, null, null, 1])"
			})

			It("returns all non-nulls", func() {
				out, _, err := prg.Eval(map[string]any{})
				Expect(err).NotTo(HaveOccurred())

				result, err := out.ConvertToNative(reflect.TypeOf([]any{}))
				Expect(err).NotTo(HaveOccurred())

				Expect(result).To(Equal([]any{
					map[ref.Val]ref.Val{types.String("key"): types.String("value")}, int64(1),
				}))
			})
		})
	})

	Describe("Pluck", func() {
		BeforeEach(func() {
			src = "pluck([{'severity': 'major'}, {'severity': 'minor'}], 'severity')"
		})

		When("the input is a valid list of objects", func() {
			It("returns an array of the values for the given key", func() {
				out, _, err := prg.Eval(map[string]any{})
				Expect(err).NotTo(HaveOccurred())

				result, err := out.ConvertToNative(reflect.TypeOf([]string{}))
				Expect(err).NotTo(HaveOccurred())

				Expect(result).To(Equal([]string{"major", "minor"}))
			})
		})
	})

	Describe("TrimPrefix", func() {
		BeforeEach(func() {
			src = "trimPrefix(value, 'group:')"
		})

		When("the input is a string", func() {
			It("returns the input without prefix", func() {
				out, _, err := prg.Eval(map[string]any{
					"value": "group:engineering",
				})
				Expect(err).NotTo(HaveOccurred())

				result, err := out.ConvertToNative(reflect.TypeOf(""))
				Expect(err).NotTo(HaveOccurred())

				Expect(result).To(Equal("engineering"))
			})
		})
	})
})
