package jvm

import (
	"github.com/756445638/lucy/src/cmd/compile/ast"
)

var (
	arrylistclassname   = "java/util/ArrayList"
	specail_method_init = "<init>"
	java_string_class   = "java/lang/String"
	java_hashmap_class  = "java/util/HashMap"
	ArrayMetas          = map[int]*ArrayMeta{}
	java_integer_class  = "java/lang/Integer"
	java_float_class    = "java/lang/Float"
	java_double_class   = "java/lang/Double"
	java_long_class     = "java/lang/Long"
)

func init() {
	ast.JvmSlotSizeHandler = func(v *ast.VariableType) uint16 {
		if v.Typ == ast.VARIABLE_TYPE_DOUBLE || ast.VARIABLE_TYPE_LONG == v.Typ {
			return 2
		} else {
			return 1
		}
	}
}
