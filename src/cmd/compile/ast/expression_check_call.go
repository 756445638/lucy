package ast

import (
	"fmt"

	"github.com/756445638/lucy/src/cmd/compile/common"
)

func (e *Expression) checkMethodCallExpression(block *Block, errs *[]error) []*VariableType {
	call := e.Data.(*ExpressionMethodCall)
	ts, es := call.Expression.check(block)
	if errsNotEmpty(es) {
		*errs = append(*errs, es...)
	}
	object, err := e.mustBeOneValueContext(ts)
	if err != nil {
		*errs = append(*errs, err)
	}
	if object.Typ == VARIABLE_TYPE_MAP {
		switch call.Name {
		case common.MAP_METHOD_KEY_EXISTS, common.MAP_METHOD_VALUE_EXISTS:
			ret := &VariableType{}
			ret.Pos = e.Pos
			ret.Typ = VARIABLE_TYPE_BOOL
			if len(call.Args) == 0 || len(call.Args) > 1 {
				*errs = append(*errs, fmt.Errorf("%s call expect one argument", errMsgPrefix(e.Pos), call.Name))
				return []*VariableType{ret}
			}
			matchkey := true
			if call.Name == common.MAP_METHOD_VALUE_EXISTS {
				matchkey = false
			}
			ts, es := call.Args[0].check(block)
			if errsNotEmpty(es) {
				*errs = append(*errs, es...)
			}
			t, err := call.Args[0].mustBeOneValueContext(ts)
			if err != nil {
				*errs = append(*errs, err)
			}
			if matchkey {
				if false == object.Map.K.Equal(t) {
					*errs = append(*errs, fmt.Errorf("%s cannot use '%s' as '%s'",
						errMsgPrefix(e.Pos), t.TypeString(), object.Map.K.TypeString()))
				}
			} else {
				if false == object.Map.V.Equal(t) {
					*errs = append(*errs, fmt.Errorf("%s cannot use '%s' as '%s'",
						errMsgPrefix(e.Pos), t.TypeString(), object.Map.V.TypeString()))
				}
			}
			return []*VariableType{ret}
		case common.MAP_METHOD_REMOVE:
			ret := &VariableType{}
			ret.Pos = e.Pos
			ret.Typ = VARIABLE_TYPE_VOID
			if len(call.Args) == 0 {
				*errs = append(*errs, fmt.Errorf("%s remove expect at last on argement",
					errMsgPrefix(e.Pos), e.Pos))
			}
			for _, v := range call.Args {
				ts, es := v.check(block)
				if errsNotEmpty(es) {
					*errs = append(*errs, es...)
				}
				for _, t := range ts {
					if object.Map.K.Equal(t) == false {
						*errs = append(*errs, fmt.Errorf("%s cannot use '%s' as '%s'",
							errMsgPrefix(e.Pos), t.TypeString(), object.Map.K.TypeString()))
					}
				}
			}
			return []*VariableType{ret}
		case common.MAP_METHOD_REMOVEALL:
			ret := &VariableType{}
			ret.Pos = e.Pos
			ret.Typ = VARIABLE_TYPE_VOID
			if len(call.Args) > 0 {
				*errs = append(*errs, fmt.Errorf("%s removeAll expect no arguments",
					errMsgPrefix(e.Pos), e.Pos))
			}
			return []*VariableType{ret}
		case common.MAP_METHOD_SIZE:
			ret := &VariableType{}
			ret.Pos = e.Pos
			ret.Typ = VARIABLE_TYPE_INT
			if len(call.Args) > 0 {
				*errs = append(*errs, fmt.Errorf("%s too many argument to call size'", errMsgPrefix(e.Pos)))
			}
			return []*VariableType{ret}
		default:
			*errs = append(*errs, fmt.Errorf("%s unkown call '%s' on map", errMsgPrefix(e.Pos), call.Name))
			return nil
		}
		return nil
	}
	if object.Typ == VARIABLE_TYPE_ARRAY {
		switch call.Name {
		case common.ARRAY_METHOD_SIZE,
			common.ARRAY_METHOD_CAP,
			common.ARRAY_METHOD_START,
			common.ARRAY_METHOD_END:
			t := &VariableType{}
			t.Typ = VARIABLE_TYPE_INT
			t.Pos = e.Pos
			if len(call.Args) > 0 {
				*errs = append(*errs, fmt.Errorf("%s too mamy argument to call,method '%s' expect no arguments",
					errMsgPrefix(e.Pos), call.Name))
			}
			return []*VariableType{t}
		case common.ARRAY_METHOD_APPEND, common.ARRAY_METHOD_APPEND_ALL:
			if len(call.Args) == 0 {
				*errs = append(*errs, fmt.Errorf("%s too few arguments to call %s,expect at least one argument",
					errMsgPrefix(e.Pos), call.Name))
			}
			for _, e := range call.Args {
				ts, es := e.check(block)
				if errsNotEmpty(es) {
					*errs = append(*errs, es...)
				}
				for _, t := range ts {
					if call.Name == common.ARRAY_METHOD_APPEND {
						if object.ArrayType.Equal(t) == false {
							*errs = append(*errs, fmt.Errorf("%s cannot use '%s' as '%s' to call method '%s'",
								errMsgPrefix(t.Pos), t.TypeString(), object.ArrayType.TypeString(), call.Name))
						}
					} else {
						if object.Equal(t) == false {
							*errs = append(*errs, fmt.Errorf("%s cannot use '%s' as '%s' to call method '%s'",
								errMsgPrefix(t.Pos), t.TypeString(), object.ArrayType.TypeString(), call.Name))
						}
					}
				}
			}
			t := object.Clone()
			t.Pos = e.Pos
			return []*VariableType{t}
		default:
			*errs = append(*errs, fmt.Errorf("%s unkown call '%s' on array", errMsgPrefix(e.Pos), call.Name))
		}
		return nil
	}
	if object.Typ != VARIABLE_TYPE_OBJECT && object.Typ != VARIABLE_TYPE_CLASS {
		*errs = append(*errs, fmt.Errorf("%s cannot make method call named '%s' on '%s'", errMsgPrefix(e.Pos), call.Name, object.TypeString()))
		return nil
	}
	args := checkExpressions(block, call.Args, errs)
	args = checkRightValuesValid(args, errs)
	f, es := object.Class.accessMethod(call.Name, e.Pos, args)
	if errsNotEmpty(es) {
		*errs = append(*errs, fmt.Errorf("%s %s", errMsgPrefix(e.Pos), err))
	} else {
		if !call.Expression.isThisIdentifierExpression() {
			*errs = append(*errs, fmt.Errorf("%s method  %s is not public", errMsgPrefix(e.Pos), call.Name))
		}
	}
	if f == nil {
		return nil
	}
	return args
}

func (e *Expression) checkFunctionCallExpression(block *Block, errs *[]error) []*VariableType {
	call := e.Data.(*ExpressionFunctionCall)
	tt, es := call.Expression.check(block)
	if errsNotEmpty(es) {
		*errs = append(*errs, es...)
	}
	t, err := e.mustBeOneValueContext(tt)
	if err != nil {
		*errs = append(*errs, err)
	}
	if t == nil {
		*errs = append(*errs, fmt.Errorf("%s %s not found", errMsgPrefix(e.Pos), call.Expression.OpName()))
		t = &VariableType{
			Typ: VARIABLE_TYPE_VOID,
			Pos: e.Pos,
		}
		return nil
	}
	if t.Typ != VARIABLE_TYPE_FUNCTION {
		*errs = append(*errs, fmt.Errorf("%s %s is not a function", errMsgPrefix(e.Pos), call.Expression.OpName()))
		t = &VariableType{
			Typ: VARIABLE_TYPE_VOID,
			Pos: e.Pos,
		}
		return []*VariableType{t}
	}
	call.Func = t.Function
	if t.Function.IsBuildin {
		return e.checkBuildinFunctionCall(block, errs, t.Function, call.Args)
	} else {
		return e.checkFunctionCall(block, errs, t.Function, call.Args)
	}
}

func (e *Expression) checkFunctionCall(block *Block, errs *[]error, f *Function, args []*Expression) []*VariableType {
	callargsTypes := checkExpressions(block, args, errs)
	callargsTypes = checkRightValuesValid(callargsTypes, errs)
	if len(callargsTypes) > len(f.Typ.ParameterList) {
		*errs = append(*errs, fmt.Errorf("%s too many paramaters to call function %s", errMsgPrefix(e.Pos), f.Name))
	}
	if len(callargsTypes) < len(f.Typ.ParameterList) && len(args) < len(f.Typ.ParameterList) {
		*errs = append(*errs, fmt.Errorf("%s too few paramaters to call function %s", errMsgPrefix(e.Pos), f.Name))
	}
	for k, v := range f.Typ.ParameterList {
		if k < len(callargsTypes) {
			if !v.Typ.TypeCompatible(callargsTypes[k]) {
				*errs = append(*errs, fmt.Errorf("%s type %s is not compatible with %s",
					errMsgPrefix(args[k].Pos),
					v.Typ.TypeString(),
					callargsTypes[k].TypeString()))
			}
		}
	}
	return f.Typ.ReturnList.retTypes(e.Pos)
}
