package expr

import (
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

func Stdlib() []cel.EnvOption {
	return []cel.EnvOption{
		Pluck(),
		Coalesce(),
		First(),
	}
}

// Pluck is a function that given a list of object, will map over those objects and return
// the values at the provided key.
func Pluck() cel.EnvOption {
	binding := func(lhs, rhs ref.Val) ref.Val {
		result := []any{}

		iter := lhs.(traits.Lister).Iterator()
		for iter.HasNext() == types.True {
			value := iter.Next()
			valueMap, err := value.ConvertToNative(reflect.TypeOf(map[string]any{}))
			if err != nil {
				panic("value is not map")
			}

			result = append(result, valueMap.(map[string]any)[rhs.Value().(string)])
		}

		reg, err := types.NewRegistry()
		if err != nil {
			panic(err)
		}

		return types.NewDynamicList(reg, result)
	}

	return cel.Function("pluck",
		cel.Overload("pluck_string_map_string_any",
			[]*cel.Type{cel.ListType(cel.MapType(cel.StringType, cel.AnyType)), cel.StringType},
			cel.AnyType,
			cel.BinaryBinding(binding),
		),
	)
}

// Coalesce removes all null values from a list.
func Coalesce() cel.EnvOption {
	binding := func(val ref.Val) ref.Val {
		result := []any{}

		iter := val.(traits.Lister).Iterator()
		for iter.HasNext() == types.True {
			value := iter.Next()
			if value != types.NullValue {
				result = append(result, value.Value())
			}
		}

		reg, err := types.NewRegistry()
		if err != nil {
			panic(err)
		}

		return types.NewDynamicList(reg, result)
	}

	return cel.Function("coalesce",
		cel.Overload("coalesce_list_list",
			[]*cel.Type{cel.ListType(cel.AnyType)},
			cel.ListType(cel.AnyType),
			cel.UnaryBinding(binding),
		),
	)
}

// First returns the first value of a list element, if it exists.
func First() cel.EnvOption {
	binding := func(val ref.Val) ref.Val {
		return val.ConvertToType(types.ListType).(traits.Indexer).Get(types.IntZero)
	}

	return cel.Function("first",
		cel.Overload("first_list_any",
			[]*cel.Type{cel.ListType(cel.AnyType)},
			cel.AnyType,
			cel.UnaryBinding(binding),
		),
	)
}
