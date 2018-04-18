package ast

import (
	"fmt"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/common"
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
	if object == nil {
		return nil
	}
	if object.Typ == VARIABLE_TYPE_PACKAGE {
		if object.Package.Block.SearchByName(call.Name) == nil {
			*errs = append(*errs, fmt.Errorf("%s function '%s' not found", errMsgPrefix(e.Pos), call.Name))
			return nil
		}
		if object.Package.Block.Funcs != nil && object.Package.Block.Funcs[call.Name] != nil {
			f := object.Package.Block.Funcs[call.Name]
			e.Typ = EXPRESSION_TYPE_FUNCTION_CALL
			functionCall := &ExpressionFunctionCall{}
			functionCall.Func = f
			functionCall.Args = call.Args
			e.Data = functionCall
			return e.checkFunctionCall(block, errs, f, &functionCall.Args)
		} else if object.Package.Block.Classes != nil && object.Package.Block.Classes[call.Name] != nil {
			//object cast
			class := object.Package.Block.Classes[call.Name]
			ret := make([]*VariableType, 1)
			ret[0] = &VariableType{}
			ret[0].Typ = VARIABLE_TYPE_OBJECT
			ret[0].Pos = e.Pos
			ret[0].Class = class
			e.Typ = EXPRESSION_TYPE_CONVERTION_TYPE
			typeConvertion := &ExpressionTypeConvertion{}
			typeConvertion.Typ = ret[0]
			if len(call.Args) >= 1 {
				typeConvertion.Expression = call.Args[0]
			}
			e.Data = typeConvertion
			if len(call.Args) != 1 {
				*errs = append(*errs, fmt.Errorf("%s cast type expect 1 argument", errMsgPrefix(e.Pos)))
				return ret
			}
			ts, es := call.Args[0].check(block)
			if errsNotEmpty(es) {
				*errs = append(*errs, es...)
			}
			t, err := call.Args[0].mustBeOneValueContext(ts)
			if err != nil {
				*errs = append(*errs, err)
			}
			if t == nil {
				return ret
			}
			if t.IsPrimitive() {
				*errs = append(*errs, fmt.Errorf("%s expression is primitive,cannot be cast to another type",
					errMsgPrefix(e.Pos)))
			}
			return ret
		} else {
			*errs = append(*errs, fmt.Errorf("%s '%s' is not a function", errMsgPrefix(e.Pos), call.Name))
			return nil
		}
		return nil
	}
	if object.Typ == VARIABLE_TYPE_MAP {
		switch call.Name {
		case common.MAP_METHOD_KEY_EXISTS:
			ret := &VariableType{}
			ret.Pos = e.Pos
			ret.Typ = VARIABLE_TYPE_BOOL
			if len(call.Args) == 0 || len(call.Args) > 1 {
				*errs = append(*errs, fmt.Errorf("%s call '%s' expect one argument",
					errMsgPrefix(e.Pos), call.Name))
				return []*VariableType{ret}
			}
			matchkey := call.Name == common.MAP_METHOD_KEY_EXISTS
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
					errMsgPrefix(e.Pos)))
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
					errMsgPrefix(e.Pos)))
			}
			return []*VariableType{ret}
		case common.MAP_METHOD_SIZE:
			ret := &VariableType{}
			ret.Pos = e.Pos
			ret.Typ = VARIABLE_TYPE_INT
			if len(call.Args) > 0 {
				*errs = append(*errs, fmt.Errorf("%s too many argument to call '%s''",
					errMsgPrefix(e.Pos), call.Name))
			}
			return []*VariableType{ret}
		default:
			*errs = append(*errs, fmt.Errorf("%s unkown call '%s' on map", errMsgPrefix(e.Pos), call.Name))
			return nil
		}
		return nil
	}
	if object.Typ == VARIABLE_TYPE_JAVA_ARRAY {
		switch call.Name {
		case common.ARRAY_METHOD_SIZE:
			t := &VariableType{}
			t.Typ = VARIABLE_TYPE_INT
			t.Pos = e.Pos
			if len(call.Args) > 0 {
				*errs = append(*errs, fmt.Errorf("%s method '%s' expect no arguments",
					errMsgPrefix(e.Pos), call.Name))
			}
			return []*VariableType{t}
		default:
			*errs = append(*errs, fmt.Errorf("%s unkown call '%s' on '%s'",
				errMsgPrefix(e.Pos), call.Name, object.TypeString()))
		}
		return nil
	}

	if object.Typ == VARIABLE_TYPE_ARRAY {
		switch call.Name {
		case common.ARRAY_METHOD_SIZE,
			common.ARRAY_METHOD_CAP,   //for debug,remove when time is right
			common.ARRAY_METHOD_START, //for debug,remove when time is right
			common.ARRAY_METHOD_END:   //for debug,remove when time is right
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
		*errs = append(*errs, fmt.Errorf("%s cannot make method call named '%s' on '%s'",
			errMsgPrefix(e.Pos), call.Name, object.TypeString()))
		return nil
	}
	call.Class = object.Class
	args := checkExpressions(block, call.Args, errs)
	args = checkRightValuesValid(args, errs)
	ms, matched, err := object.Class.accessMethod(call.Name, args, &call.Args, false)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s %v", errMsgPrefix(e.Pos), err))
		return nil
	}
	if matched {
		if false == call.Expression.isThisIdentifierExpression() &&
			ms[0].IsPublic() == false {
			*errs = append(*errs, fmt.Errorf("%s method  %s is not public", errMsgPrefix(e.Pos), call.Name))
		}
		call.Method = ms[0]
		return ms[0].Func.Typ.ReturnList.retTypes(e.Pos)
	}
	if ms == nil || len(ms) == 0 {
		*errs = append(*errs, fmt.Errorf("%s method '%s' not found", errMsgPrefix(e.Pos), call.Name))
	} else {
		*errs = append(*errs, msNotMatchError(e.Pos, call.Name, ms, args))
	}
	return nil
}