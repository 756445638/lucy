package ast

import (
	"fmt"

	"gitee.com/yuyang-fine/lucy/src/cmd/compile/jvm/cg"
)

func (e *Expression) getLeftValue(block *Block, errs *[]error) (ret *Type) {
	switch e.Type {
	case EXPRESSION_TYPE_IDENTIFIER:
		identifier := e.Data.(*ExpressionIdentifier)
		d, _ := block.searchRightValue(identifier.Name)
		if d == nil {
			*errs = append(*errs, fmt.Errorf("%s '%s' not found",
				errMsgPrefix(e.Pos), identifier.Name))
			return nil
		}
		switch d.(type) {
		case *Variable:
			if identifier.Name == THIS {
				*errs = append(*errs, fmt.Errorf("%s '%s' cannot be used as left value",
					errMsgPrefix(e.Pos), THIS))
			}
			t := d.(*Variable)
			identifier.Variable = t
			ret = identifier.Variable.Type.Clone()
			ret.Pos = e.Pos
			return ret
		default:
			*errs = append(*errs, fmt.Errorf("%s identifier named '%s' is not variable",
				errMsgPrefix(e.Pos), identifier.Name))
			return nil
		}
	case EXPRESSION_TYPE_INDEX:
		ret = e.checkIndexExpression(block, errs)
		return ret
	case EXPRESSION_TYPE_SELECTION:
		dot := e.Data.(*ExpressionSelection)
		t, es := dot.Expression.checkSingleValueContextExpression(block)
		if errorsNotEmpty(es) {
			*errs = append(*errs, es...)
		}
		if t == nil {
			return nil
		}
		switch t.Type {
		case VARIABLE_TYPE_OBJECT:
			field, err := t.Class.accessField(dot.Name, false)
			if err != nil {
				*errs = append(*errs, fmt.Errorf("%s %v", errMsgPrefix(e.Pos), err))
			}
			dot.Field = field
			if field != nil {
				if field.IsStatic() {
					*errs = append(*errs, fmt.Errorf("%s field '%s' is static,should access by class",
						errMsgPrefix(e.Pos), dot.Name))
				}
				// not this and private
				if dot.Expression.isThis() == false && field.IsPrivate() {
					*errs = append(*errs, fmt.Errorf("%s field '%s' is private",
						errMsgPrefix(e.Pos), dot.Name))
				}
				ret = field.Type.Clone()
				ret.Pos = e.Pos
				return ret
			}
			return nil
		case VARIABLE_TYPE_CLASS:
			field, err := t.Class.accessField(dot.Name, false)
			if err != nil {
				*errs = append(*errs, fmt.Errorf("%s %v", errMsgPrefix(e.Pos), err))
			}
			dot.Field = field
			if field != nil {
				if field.IsStatic() == false {
					*errs = append(*errs, fmt.Errorf("%s field '%s' is not static,should access by instance",
						errMsgPrefix(e.Pos), dot.Name))
				}
				ret = field.Type.Clone()
				ret.Pos = e.Pos
				return ret
			}
			return nil
		case VARIABLE_TYPE_PACKAGE:
			variable, exists := t.Package.Block.NameExists(dot.Name)
			if exists == false {
				*errs = append(*errs, fmt.Errorf("%s '%s.%s' not found",
					errMsgPrefix(e.Pos), t.Package.Name, dot.Name))
				return nil
			}
			if vd, ok := variable.(*Variable); ok && vd != nil {
				if vd.AccessFlags&cg.ACC_FIELD_PUBLIC == 0 {
					*errs = append(*errs, fmt.Errorf("%s '%s.%s' is private",
						errMsgPrefix(e.Pos), t.Package.Name, dot.Name))
				}
				dot.PackageVariable = vd
				ret = vd.Type.Clone()
				ret.Pos = e.Pos
				return ret
			} else {
				*errs = append(*errs, fmt.Errorf("%s '%s.%s' is not variable",
					errMsgPrefix(e.Pos), t.Package.Name, dot.Name))
				return nil
			}
		default:
			*errs = append(*errs, fmt.Errorf("%s '%s' cannot be used as left value",
				errMsgPrefix(e.Pos),
				dot.Expression.OpName()))
			return nil
		}
	default:
		*errs = append(*errs, fmt.Errorf("%s '%s' cannot be used as left value",
			errMsgPrefix(e.Pos),
			e.OpName()))
		return nil
	}
	return nil
}
