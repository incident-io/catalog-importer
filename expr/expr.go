package expr

import (
	"context"
	"fmt"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/ext"
	"github.com/pkg/errors"
)

// Compile takes a CEL expression and compiles it with the standard environment. We don't
// have anything in the standard environment just yet, but this ensures we instantiate the
// CEL VM consistently, wherever we might use it.
func Compile(source string) (cel.Program, error) {
	env, err := cel.NewEnv(Stdlib(), ext.Strings())
	if err != nil {
		return nil, errors.Wrap(err, "building CEL environment")
	}

	ast, issues := env.Parse(source)
	if err := issues.Err(); err != nil {
		return nil, errors.Wrap(err, "compiling CEL expression")
	}

	prg, err := env.Program(ast)
	if err != nil {
		return nil, errors.Wrap(err, "building CEL program")
	}

	return prg, nil
}

// Eval evaluates the given program against the scope, returning a value that matches the
// type requested via the generic ReturnType parameter.
func Eval[ReturnType any](ctx context.Context, prg cel.Program, scope map[string]any) (result ReturnType, err error) {
	// It's useful to be able to refer directly to the top-level scope via a non-escaped
	// variable name, as it's otherwise impossible to access a scope variable that has
	// whitespace (e.g. "Name of thing").
	//
	// By adding a reference at _, this becomes possible via `_["Name of thing"]`.
	scope["_"] = scope

	out, _, err := prg.ContextEval(ctx, scope)
	if err != nil {
		return
	}

	outResult, err := out.ConvertToNative(reflect.TypeOf(result))
	if err != nil {
		return
	}

	result, ok := outResult.(ReturnType)
	if !ok {
		return result, fmt.Errorf("could not convert result of %T to %T", outResult, result)
	}

	return result, nil
}
