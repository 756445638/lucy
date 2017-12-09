package ast

import (
	"fmt"
)

const (
	VARIABLE_TYPE_BOOL = iota
	VARIABLE_TYPE_BYTE
	VARIABLE_TYPE_SHORT
	VARIABLE_TYPE_INT
	VARIABLE_TYPE_LONG
	VARIABLE_TYPE_FLOAT
	VARIABLE_TYPE_DOUBLE
	VARIABLE_TYPE_CHAR
	VARIABLE_TYPE_STRING
	VARIALBE_TYPE_FUNCTION
	VARIALBE_TYPE_ENUM  //enum
	VARIABLE_TYPE_CLASS //new Person()
	VARIABLE_TYPE_NULL  //null
	VARIABLE_TYPE_ARRAY // []int
	VARIABLE_TYPE_NAME  // naming should search for declaration
	VARIABLE_TYPE_VOID
)

type VariableType struct {
	Pos             *Pos
	Typ             int
	Name            string // Lname.Rname
	CombinationType *VariableType
	FunctionType    *FunctionType
	Class           *Class
}

func (v *VariableType) Signature() string {
	switch v.Typ {
	case VARIABLE_TYPE_BOOL:
		return "Z"
	case VARIABLE_TYPE_BYTE:
		return "B"
	case VARIABLE_TYPE_INT:
		return "I"
	case VARIABLE_TYPE_LONG:
		return "J"
	case VARIABLE_TYPE_FLOAT:
		return "F"
	case VARIABLE_TYPE_DOUBLE:
		return "D"
	case VARIABLE_TYPE_VOID:
		return "V"
	case VARIABLE_TYPE_SHORT:
		return "S"
	case VARIABLE_TYPE_ARRAY:
		return "[" + v.CombinationType.Signature()
	case VARIABLE_TYPE_STRING:
		return "Ljava/lang/String"
	}
	panic("unhandle type signature")
}

//func (t *VariableType) matchExpression(e *Expression) bool {
//	return false
//}

func (t *VariableType) typeCompatible(t2 *VariableType) bool {
	if t.Equal(t2) {
		return true
	}
	return false
}

func (t *VariableType) constValueValid(e *Expression) {

}

//assign some simple expression
func (t *VariableType) assignExpression(p *Package, e *Expression) (data interface{}, err error) {
	switch t.Typ {
	case VARIABLE_TYPE_BOOL:
		if e.Typ == EXPRESSION_TYPE_BOOL {
			data = e.Data.(bool)
			return
		}
	case VARIALBE_TYPE_ENUM:
		if e.Typ == EXPRESSION_TYPE_IDENTIFIER {
			if _, ok := p.EnumNames[e.Data.(string)]; ok {
				data = p.EnumNames[e.Data.(string)]
			}
		}
	case VARIABLE_TYPE_BYTE:
		if e.Typ == EXPRESSION_TYPE_BYTE {
			data = e.Data.(byte)
			return
		}
	case VARIABLE_TYPE_INT:
		if e.Typ == EXPRESSION_TYPE_BYTE {
			data = int64(e.Data.(byte))
			return
		} else if e.Typ == EXPRESSION_TYPE_INT {
			data = e.Data.(int64)
			return
		}
	case VARIABLE_TYPE_FLOAT:
		if e.Typ == EXPRESSION_TYPE_BYTE {
			data = int64(e.Data.(byte))
			return
		} else if e.Typ == EXPRESSION_TYPE_INT {
			data = e.Data.(int64)
			return
		} else if EXPRESSION_TYPE_FLOAT == e.Typ {
			data = e.Data.(float64)
			return
		}
	case VARIABLE_TYPE_STRING:
		if e.Typ == EXPRESSION_TYPE_STRING {
			data = e.Data.(string)
			return
		}
	case VARIABLE_TYPE_CLASS:
		if e.Typ == EXPRESSION_TYPE_NULL {
			return nil, nil // null pointer
		}
	}
	err = fmt.Errorf("can`t covert type accroding to type")
	return
}

////把树型转化为可读字符串
//func (c *CombinationType) TypeString(ret *string) {
//	if c.Typ == COMBINATION_TYPE_ARRAY {
//		*ret += "[]"
//	}
//	c.Combination.TypeString(ret)
//}

//可读的类型信息
func (v *VariableType) TypeString(ret *string) {
	switch v.Typ {
	case VARIABLE_TYPE_BOOL:
		*ret = "bool"
	case VARIABLE_TYPE_BYTE:
		*ret = "byte"
	case VARIABLE_TYPE_INT:
		*ret = "int"
	case VARIABLE_TYPE_FLOAT:
		*ret = "float"
	case VARIALBE_TYPE_FUNCTION:
		*ret = "function"
	case VARIABLE_TYPE_CLASS:
		*ret = v.Name
	case VARIALBE_TYPE_ENUM:
		*ret = v.Name + "(enum)"
	case VARIABLE_TYPE_ARRAY:
		*ret += "[]"
		v.CombinationType.TypeString(ret)
	}
}

func (v *VariableType) Equal(e *VariableType) bool {
	if v.Typ != e.Typ {
		return false
	}
	if v.CombinationType.Typ != e.CombinationType.Typ {
		return false
	}
	return v.CombinationType.Equal(e.CombinationType)
}
