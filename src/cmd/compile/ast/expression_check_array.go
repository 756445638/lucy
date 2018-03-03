package ast

import (
	"fmt"
)

//in array type must eqaul
func (e *Expression) checkArray(block *Block, errs *[]error) *VariableType {
	arr := e.Data.(*ExpressionArrayLiteral)
	if arr.Typ == nil && (arr.Expressions == nil || len(arr.Expressions) == 0) {
		*errs = append(*errs, fmt.Errorf("%s array literal has no type, no expression,cannot inference it`s type ", errMsgPrefix(e.Pos)))
		return nil
	}
	notyp := true
	if arr.Typ != nil {
		notyp = false
		arr.Typ.resolve(block)
	}
	for _, v := range arr.Expressions {
		ts, es := v.check(block)
		if errsNotEmpty(es) {
			*errs = append(*errs, es...)
		}
		if ts != nil {
			arr.Length += len(ts)
		}
		for _, t := range ts {
			if notyp && arr.Typ == nil {
				tt := t.Clone()
				tt.Pos = e.Pos
				arr.Typ = &VariableType{}
				arr.Typ.Typ = VARIABLE_TYPE_ARRAY_INSTANCE
				arr.Typ.ArrayType = tt
			}
			if arr.Typ.ArrayType.Equal(t) == false {
				if notyp {
					*errs = append(*errs, fmt.Errorf("%s array literal mix up '%s' and '%s'",
						errMsgPrefix(t.Pos), arr.Typ.ArrayType.TypeString(), t.TypeString()))
				} else {
					*errs = append(*errs, fmt.Errorf("%s cannot use '%s' as '%s'",
						errMsgPrefix(t.Pos), t.TypeString(), arr.Typ.ArrayType.TypeString()))
				}
			}
		}
	}
	if arr.Typ == nil {
		return nil
	}
	arr.Typ.actionNeedBeenDoneWhenDescribeVariable()
	tt := arr.Typ.Clone()
	tt.Pos = e.Pos
	return tt
}
