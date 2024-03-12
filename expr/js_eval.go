package expr

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/pkg/errors"
	"github.com/robertkrimen/otto"
	underscore "github.com/robertkrimen/otto/underscore"
)

var vm *otto.Otto

func init() {

	underscore.Enable()

	// Create a Javascript virtual machine that we'll use for evaluating the source
	// expression. We must be very careful: this is executing code on behalf of others, so
	// comes with all normal warnings.
	vm = otto.New()
	vm.Interrupt = make(chan func(), 1)

}

// EvaluateJavascript can evaluate a source Javascript program having set the given
// subject into the `$` variable.
func EvaluateJavascript(ctx context.Context, source string, subject any, logger kitlog.Logger) (result otto.Value, err error) {
	var halted bool
	defer func() {
		if caught := recover(); caught != nil {
			if halted {
				err = errors.Wrap(errors.New("timed out executing Javascript code"), err.Error())
			} else {
				panic(caught) // it wasn't our interrupt handler, repanic
			}
		}
	}()

	if vm == nil {
		panic("Javascript virtual machine not initialised")
	}

	// Start a new function bounded context.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// If we haven't finished execution after our timeout, we trigger the interrupt handler.
	SafelyGo(func() {
		select {
		case <-time.After(250 * time.Millisecond):
			vm.Interrupt <- func() {
				panic("timed out executing Javascript")
			}
		case <-ctx.Done():
			// do nothing, we finished executing
		}
	})

	// Set the subject of the expression in a variable called $ as a simple handle to access
	// everything.
	_ = vm.Set("$", subject)

	// Evaluate the source (eg. the script) against the subject, set above.
	outResult, err := vm.Run(source)
	if err != nil {
		// If we've failed to evaluate an expression, let's continue on, but give them some good debug info.
		level.Debug(logger).Log("msg", fmt.Sprintf("Could not evaluate expression \"%s\": %s. Returning nil", source, string(err.Error())))
		return outResult, nil
	}

	return outResult, nil

}

func EvaluateArray[ReturnType any](ctx context.Context, source string, subject any, logger kitlog.Logger) ([]ReturnType, error) {
	resultValues := []ReturnType{}

	result, err := EvaluateJavascript(ctx, source, subject, logger)
	if err != nil {
		return resultValues, errors.Wrap(err, "evaluating array value")
	}

	// Although we've parameterised ReturnType in both EvaluateArray and EvaluateSingleValue,
	// if the caller expects multi-value results, we need to treat the return value differently
	// than if it's a single value. Hence why we need to loop through our JS evaluated values,
	// and explicitly return a slice of these type-checked results.
	evaluatedValues := []otto.Value{}

	if result.IsObject() {
		switch result.Object().Class() {
		case "GoSlice", "Array":
			for _, key := range result.Object().Keys() {
				// This should always work, as we just asked for the available keys.
				element, err := result.Object().Get(key)
				if err != nil {
					return resultValues, err
				}

				evaluatedValues = append(evaluatedValues, element)
			}
		}
	} else {
		// Even if the input doesn't seem to be multi-value,
		// let's iterate and return an array as expected.
		evaluatedValues = append(evaluatedValues, result)
	}

	// We parsed our JS successfully, and have multiple values, as expected.
	// Now parse each nested value and return the final slice.
	for _, evaluatedValue := range evaluatedValues {
		resultValue, err := EvaluateResultType[ReturnType](ctx, source, evaluatedValue)
		if err != nil {
			return resultValues, nil
		}
		resultValues = append(resultValues, resultValue)
	}

	return resultValues, nil
}

func EvaluateSingleValue[ReturnType any](ctx context.Context, source string, subject any, logger kitlog.Logger) (ReturnType, error) {
	var resultValue ReturnType
	result, err := EvaluateJavascript(ctx, source, subject, logger)
	if err != nil {
		return resultValue, errors.Wrap(err, "evaluating single value")
	}

	resultValue, err = EvaluateResultType[ReturnType](ctx, source, result)
	if err != nil {
		return resultValue, err
	}

	return resultValue, nil
}

func EvaluateResultType[ReturnType any](ctx context.Context, source string, result otto.Value) (ReturnType, error) {
	var resultValue ReturnType
	var ok bool
	switch {
	case result.IsBoolean():
		resultBool, err := result.ToBoolean()
		if err != nil {
			return resultValue, err
		}

		// This is a pattern we'll employ in each of the checks below
		// to see if our result value matches the expected ReturnType.
		// It's slightly gross, but does the trick.
		typeAgnosticResult := any(resultBool)

		// If OK, this is supported by Bool.
		resultValue, ok := typeAgnosticResult.(ReturnType)
		if !ok {
			// In bool's case, if not ok, try the value again as a string.
			boolValue := fmt.Sprintf("%v", resultBool)
			typeAgnosticResult := any(boolValue)
			resultValue, ok = typeAgnosticResult.(ReturnType)
			if !ok {
				return resultValue, fmt.Errorf("could not convert result of bool to %T", resultValue)
			}
		}

		return resultValue, nil

	case result.IsNumber():
		resultInt, err := strconv.Atoi(fmt.Sprintf("%v", result))
		if err != nil {
			return resultValue, err
		}

		typeAgnosticResult := any(resultInt)

		// If OK, this is supported by Number.
		resultValue, ok = typeAgnosticResult.(ReturnType)
		if !ok {
			// In number's case, if not ok, try the value again as a string.
			intValue := fmt.Sprintf("%v", resultInt)
			typeAgnosticResult := any(intValue)
			resultValue, ok = typeAgnosticResult.(ReturnType)
			if !ok {
				return resultValue, fmt.Errorf("could not convert result of int to %T", resultValue)
			}
		}

		return resultValue, nil

	case result.IsString():
		resultString, err := result.ToString()
		if err != nil {
			return resultValue, err
		}

		stringValue := fmt.Sprintf("%v", resultString)
		typeAgnosticResult := any(stringValue)

		// If OK, this is supported by String.
		resultValue, ok = typeAgnosticResult.(ReturnType)
		if !ok {
			return resultValue, fmt.Errorf("could not convert result of string to %T", resultValue)
		}

	case result.IsUndefined():
		// do nothing, undefined gets skipped

	default:
		fmt.Fprintf(os.Stderr, "\n  Unsupported Javascript value type found by expression %s: %s.\n", source, map[string]any{
			"result": result,
		})
		return resultValue, nil
	}

	return resultValue, nil
}

func SafelyGo(do func()) {
	go func() {
		defer func() {
			recover()
		}()

		do()
	}()
}
