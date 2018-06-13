package ast

import (
	"fmt"

	"gitee.com/yuyang-fine/lucy/src/cmd/compile/jvm/cg"
)

func (e *Expression) checkColonAssignExpression(block *Block, errs *[]error) {
	bin := e.Data.(*ExpressionBinary)
	var names []*Expression
	if bin.Left.Type == EXPRESSION_TYPE_IDENTIFIER {
		names = append(names, bin.Left)
	} else if bin.Left.Type == EXPRESSION_TYPE_LIST {
		names = bin.Left.Data.([]*Expression)
	} else {
		*errs = append(*errs, fmt.Errorf("%s no names on the left", errMsgPrefix(e.Pos)))
		return
	}
	noErr := true
	values := bin.Right.Data.([]*Expression)
	ts := checkRightValuesValid(checkExpressions(block, values, errs), errs)
	if len(names) != len(ts) {
		*errs = append(*errs, fmt.Errorf("%s cannot assign %d values to %d destinations",
			errMsgPrefix(e.Pos),
			len(ts),
			len(names)))
		noErr = false
	}
	var err error
	noNewVariable := true
	declareVariableExpression := &ExpressionDeclareVariable{}
	declareVariableExpression.Values = values
	for k, v := range names {
		if v.Type != EXPRESSION_TYPE_IDENTIFIER {
			*errs = append(*errs, fmt.Errorf("%s not a name on the left,but '%s'",
				errMsgPrefix(v.Pos), v.OpName()))
			noErr = false
			continue
		}
		identifier := v.Data.(*ExpressionIdentifier)
		if identifier.Name == NO_NAME_IDENTIFIER {
			vd := &VariableDefinition{}
			vd.Name = identifier.Name
			declareVariableExpression.Variables = append(declareVariableExpression.Variables, vd)
			declareVariableExpression.IfDeclaredBefore = append(declareVariableExpression.IfDeclaredBefore, false)
			continue
		}
		var variableType *VariableType
		if k < len(ts) && ts[k] != nil {
			variableType = ts[k]
		}
		if variable, ok := block.Variables[identifier.Name]; ok {
			if variableType != nil {
				if variable.Type.Equal(errs, ts[k]) == false {
					*errs = append(*errs, fmt.Errorf("%s cannot assign '%s' to '%s'",
						errMsgPrefix(ts[k].Pos),
						variable.Type.TypeString(),
						ts[k].TypeString()))
					noErr = false
				}
			}
			identifier.Variable = variable
			declareVariableExpression.Variables = append(declareVariableExpression.Variables, variable)
			declareVariableExpression.IfDeclaredBefore = append(declareVariableExpression.IfDeclaredBefore, true)
		} else { // should be no error
			noNewVariable = false
			vd := &VariableDefinition{}
			if k < len(ts) {
				vd.Type = ts[k]
			}
			vd.Name = identifier.Name
			vd.Pos = v.Pos
			vd.Type = variableType
			if vd.Type == nil { // still cannot have type,we can have a void,that`s ok
				vd.Type = &VariableType{}
				vd.Type.Type = VARIABLE_TYPE_VOID
				vd.Type.Pos = v.Pos
			}
			err = block.Insert(vd.Name, v.Pos, vd)
			identifier.Variable = vd
			if err != nil {
				*errs = append(*errs, err)
				noErr = false
				continue
			}
			declareVariableExpression.Variables = append(declareVariableExpression.Variables, vd)
			declareVariableExpression.IfDeclaredBefore = append(declareVariableExpression.IfDeclaredBefore, false)
			if e.IsPublic { // only use when is is global
				vd.AccessFlags |= cg.ACC_FIELD_PUBLIC
			}
		}
	}
	if noNewVariable {
		*errs = append(*errs, fmt.Errorf("%s no new variables to create", errMsgPrefix(e.Pos)))
		noErr = false
	}
	if noErr == false {
		return
	}
	// no error,rewrite data
	e.Data = declareVariableExpression
}

func (e *Expression) checkOpAssignExpression(block *Block, errs *[]error) (t *VariableType) {
	bin := e.Data.(*ExpressionBinary)
	t1 := bin.Left.getLeftValue(block, errs)
	bin.Left.ExpressionValue = t1
	t2, es := bin.Right.checkSingleValueContextExpression(block)
	if errorsNotEmpty(es) {
		*errs = append(*errs, es...)
	}
	if t1 == nil || t2 == nil {
		return
	}
	ret := t1.Clone()
	ret.Pos = e.Pos
	/*
		var  s string;
		s += "11111111";
	*/
	if t1.Type == VARIABLE_TYPE_STRING {
		if t2.Type != VARIABLE_TYPE_STRING || (e.Type != EXPRESSION_TYPE_PLUS_ASSIGN) {
			*errs = append(*errs, fmt.Errorf("%s cannot apply algorithm '%s' on string and '%s'",
				errMsgPrefix(e.Pos),
				e.OpName(),
				t2.TypeString()))
		}
		return ret
	}
	//number
	if e.Type == EXPRESSION_TYPE_PLUS_ASSIGN ||
		e.Type == EXPRESSION_TYPE_MINUS_ASSIGN ||
		e.Type == EXPRESSION_TYPE_MUL_ASSIGN ||
		e.Type == EXPRESSION_TYPE_DIV_ASSIGN ||
		e.Type == EXPRESSION_TYPE_MOD_ASSIGN {
		if t1.Equal(errs, t2) {
			return ret
		}
		if t1.IsInteger() && t2.IsInteger() && bin.Right.IsLiteral() {
			bin.Right.ConvertToNumber(t1.Type)
			return ret
		}
		if t1.IsFloat() && t2.IsFloat() && bin.Right.IsLiteral() {
			bin.Right.ConvertToNumber(t1.Type)
			return ret
		}

	}
	if e.Type == EXPRESSION_TYPE_AND_ASSIGN ||
		e.Type == EXPRESSION_TYPE_OR_ASSIGN ||
		e.Type == EXPRESSION_TYPE_XOR_ASSIGN {
		if t1.IsInteger() && t1.Equal(errs, t2) {
			return ret
		}
	}
	if e.Type == EXPRESSION_TYPE_LSH_ASSIGN ||
		e.Type == EXPRESSION_TYPE_RSH_ASSIGN {
		if t1.IsInteger() && t2.IsInteger() {
			if t2.Type == VARIABLE_TYPE_LONG {
				bin.Right.ConvertToNumber(VARIABLE_TYPE_INT)
			}
			return ret
		}
	}

	*errs = append(*errs, fmt.Errorf("%s cannot apply algorithm '%s' on '%s' and '%s'",
		errMsgPrefix(e.Pos),
		e.OpName(),
		t1.TypeString(),
		t2.TypeString()))

	return ret
}

func (e *Expression) checkAssignExpression(block *Block, errs *[]error) *VariableType {
	bin := e.Data.(*ExpressionBinary)
	lefts := make([]*Expression, 1)
	if bin.Left.Type == EXPRESSION_TYPE_LIST {
		lefts = bin.Left.Data.([]*Expression)
	} else {
		lefts[0] = bin.Left
		bin.Left = &Expression{}
		bin.Left.Type = EXPRESSION_TYPE_LIST
		bin.Left.Data = lefts // rewrite to list anyway
	}
	values := bin.Right.Data.([]*Expression)
	valueTypes := checkRightValuesValid(
		checkExpressions(block, values, errs),
		errs)
	leftTypes := []*VariableType{}

	for _, v := range lefts {
		if v.Type == EXPRESSION_TYPE_IDENTIFIER {
			name := v.Data.(*ExpressionIdentifier)
			if name.Name == NO_NAME_IDENTIFIER { // skip "_"
				leftTypes = append(leftTypes, nil) // this is no assign situation
				continue
			}
		}
		t := v.getLeftValue(block, errs)
		v.ExpressionValue = t
		leftTypes = append(leftTypes, t) // append even if it`s nil
	}
	convertLiteralExpressionsToNeeds(values, leftTypes, valueTypes)
	bin.Left.ExpressionMultiValues = leftTypes
	if len(lefts) != len(valueTypes) { //expression length compare with value types is more appropriate
		*errs = append(*errs, fmt.Errorf("%s cannot assign %d value to %d detinations",
			errMsgPrefix(e.Pos),
			len(valueTypes),
			len(lefts)))
	}
	for k, v := range leftTypes {
		if v == nil { // get left value error or "_"
			continue
		}
		if k >= len(valueTypes) {
			continue
		}
		if valueTypes[k] == nil {
			continue
		}
		//fmt.Println(leftTypes[k].TypeString(), valueTypes[k].TypeString())
		if false == leftTypes[k].Equal(errs, valueTypes[k]) {
			*errs = append(*errs, fmt.Errorf("%s cannot assign '%s' to '%s'",
				errMsgPrefix(e.Pos),
				valueTypes[k].TypeString(), leftTypes[k].TypeString()))
		}

	}
	voidReturn := mkVoidType(e.Pos)
	if len(leftTypes) > 1 || len(leftTypes) == 0 {
		return voidReturn
	}
	if leftTypes[0] == nil {
		return voidReturn
	}
	// here is safe
	tt := leftTypes[0].Clone()
	tt.Pos = e.Pos
	return tt
}
