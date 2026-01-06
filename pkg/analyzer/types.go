package analyzer

import (
	"fmt"
	"strings"
)

// TypeKind represents the kind of type
type TypeKind int

const (
	TypeUnknown TypeKind = iota
	TypeInteger
	TypeLong
	TypeSingle
	TypeDouble
	TypeString
	TypeBoolean
	TypeJSON
	TypePointer
	TypeChannel
	TypeArray
	TypeSlice
	TypeVoid
	TypeAny       // For Go interface{} interop
	TypeFunction
	TypeSub
)

// Type represents a DBasic type
type Type struct {
	Kind        TypeKind
	Name        string      // Original type name
	ElementType *Type       // For pointers, channels, arrays
	ArraySize   int         // For fixed-size arrays (-1 for dynamic)
	ParamTypes  []*Type     // For function/sub types
	ReturnTypes []*Type     // For function types
}

// Predefined types
var (
	IntegerType = &Type{Kind: TypeInteger, Name: "INTEGER"}
	LongType    = &Type{Kind: TypeLong, Name: "LONG"}
	SingleType  = &Type{Kind: TypeSingle, Name: "SINGLE"}
	DoubleType  = &Type{Kind: TypeDouble, Name: "DOUBLE"}
	StringType  = &Type{Kind: TypeString, Name: "STRING"}
	BooleanType = &Type{Kind: TypeBoolean, Name: "BOOLEAN"}
	JSONType    = &Type{Kind: TypeJSON, Name: "JSON"}
	VoidType    = &Type{Kind: TypeVoid, Name: "VOID"}
	AnyType     = &Type{Kind: TypeAny, Name: "ANY"}
)

// TypeFromName returns a Type for the given type name
func TypeFromName(name string) *Type {
	switch strings.ToUpper(name) {
	case "INTEGER":
		return IntegerType
	case "LONG":
		return LongType
	case "SINGLE":
		return SingleType
	case "DOUBLE":
		return DoubleType
	case "STRING":
		return StringType
	case "BOOLEAN":
		return BooleanType
	case "JSON":
		return JSONType
	case "VOID":
		return VoidType
	case "ANY":
		return AnyType
	default:
		return nil
	}
}

// NewPointerType creates a new pointer type
func NewPointerType(elem *Type) *Type {
	return &Type{
		Kind:        TypePointer,
		Name:        "POINTER TO " + elem.String(),
		ElementType: elem,
	}
}

// NewChannelType creates a new channel type
func NewChannelType(elem *Type) *Type {
	return &Type{
		Kind:        TypeChannel,
		Name:        "CHAN OF " + elem.String(),
		ElementType: elem,
	}
}

// NewArrayType creates a new array type
func NewArrayType(elem *Type, size int) *Type {
	return &Type{
		Kind:        TypeArray,
		Name:        fmt.Sprintf("%s(%d)", elem.String(), size),
		ElementType: elem,
		ArraySize:   size,
	}
}

// NewSliceType creates a new slice type (dynamic array)
func NewSliceType(elem *Type) *Type {
	return &Type{
		Kind:        TypeSlice,
		Name:        elem.String() + "()",
		ElementType: elem,
		ArraySize:   -1,
	}
}

// NewFunctionType creates a new function type
func NewFunctionType(params []*Type, returns []*Type) *Type {
	return &Type{
		Kind:        TypeFunction,
		ParamTypes:  params,
		ReturnTypes: returns,
	}
}

// NewSubType creates a new sub type (no return)
func NewSubType(params []*Type) *Type {
	return &Type{
		Kind:       TypeSub,
		ParamTypes: params,
	}
}

// String returns the string representation of the type
func (t *Type) String() string {
	if t == nil {
		return "UNKNOWN"
	}

	switch t.Kind {
	case TypePointer:
		return "POINTER TO " + t.ElementType.String()
	case TypeChannel:
		return "CHAN OF " + t.ElementType.String()
	case TypeArray:
		return fmt.Sprintf("%s(%d)", t.ElementType.String(), t.ArraySize)
	case TypeSlice:
		return t.ElementType.String() + "()"
	case TypeFunction:
		var params, rets []string
		for _, p := range t.ParamTypes {
			params = append(params, p.String())
		}
		for _, r := range t.ReturnTypes {
			rets = append(rets, r.String())
		}
		return fmt.Sprintf("FUNCTION(%s) AS (%s)", strings.Join(params, ", "), strings.Join(rets, ", "))
	case TypeSub:
		var params []string
		for _, p := range t.ParamTypes {
			params = append(params, p.String())
		}
		return fmt.Sprintf("SUB(%s)", strings.Join(params, ", "))
	default:
		return t.Name
	}
}

// GoType returns the Go type equivalent
func (t *Type) GoType() string {
	if t == nil {
		return "interface{}"
	}

	switch t.Kind {
	case TypeInteger:
		return "int32"
	case TypeLong:
		return "int64"
	case TypeSingle:
		return "float32"
	case TypeDouble:
		return "float64"
	case TypeString:
		return "string"
	case TypeBoolean:
		return "bool"
	case TypeJSON:
		return "map[string]interface{}"
	case TypePointer:
		return "*" + t.ElementType.GoType()
	case TypeChannel:
		return "chan " + t.ElementType.GoType()
	case TypeArray:
		return fmt.Sprintf("[%d]%s", t.ArraySize, t.ElementType.GoType())
	case TypeSlice:
		return "[]" + t.ElementType.GoType()
	case TypeVoid:
		return ""
	case TypeAny:
		return "interface{}"
	default:
		return "interface{}"
	}
}

// IsNumeric returns true if the type is numeric
func (t *Type) IsNumeric() bool {
	switch t.Kind {
	case TypeInteger, TypeLong, TypeSingle, TypeDouble:
		return true
	}
	return false
}

// IsInteger returns true if the type is an integer type
func (t *Type) IsInteger() bool {
	return t.Kind == TypeInteger || t.Kind == TypeLong
}

// IsFloat returns true if the type is a floating-point type
func (t *Type) IsFloat() bool {
	return t.Kind == TypeSingle || t.Kind == TypeDouble
}

// IsCompatibleWith checks if this type is compatible with another
func (t *Type) IsCompatibleWith(other *Type) bool {
	if t == nil || other == nil {
		return false
	}

	// Same type
	if t.Kind == other.Kind {
		if t.Kind == TypePointer || t.Kind == TypeChannel {
			return t.ElementType.IsCompatibleWith(other.ElementType)
		}
		if t.Kind == TypeArray || t.Kind == TypeSlice {
			return t.ElementType.IsCompatibleWith(other.ElementType)
		}
		return true
	}

	// Numeric promotion
	if t.IsNumeric() && other.IsNumeric() {
		return true
	}

	// Any type is compatible with everything
	if t.Kind == TypeAny || other.Kind == TypeAny {
		return true
	}

	return false
}

// PromoteNumeric returns the promoted type for numeric operations
func PromoteNumeric(t1, t2 *Type) *Type {
	if !t1.IsNumeric() || !t2.IsNumeric() {
		return nil
	}

	// If either is double, result is double
	if t1.Kind == TypeDouble || t2.Kind == TypeDouble {
		return DoubleType
	}

	// If either is single, result is single
	if t1.Kind == TypeSingle || t2.Kind == TypeSingle {
		return SingleType
	}

	// If either is long, result is long
	if t1.Kind == TypeLong || t2.Kind == TypeLong {
		return LongType
	}

	// Both are integer
	return IntegerType
}

// DefaultValue returns the default value expression for a type
func (t *Type) DefaultValue() string {
	switch t.Kind {
	case TypeInteger, TypeLong:
		return "0"
	case TypeSingle, TypeDouble:
		return "0.0"
	case TypeString:
		return `""`
	case TypeBoolean:
		return "false"
	case TypeJSON:
		return "make(map[string]interface{})"
	case TypePointer:
		return "nil"
	case TypeChannel:
		return "nil"
	case TypeSlice:
		return "nil"
	default:
		return "nil"
	}
}
