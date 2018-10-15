package ast

import "fmt"

func (e *Expression) checkOpAssignExpression(block *Block, errs *[]error) (t *Type) {
	bin := e.Data.(*ExpressionBinary)
	if bin.Left.Type == ExpressionTypeList {
		list := bin.Left.Data.([]*Expression)
		if len(list) > 1 {
			*errs = append(*errs,
				fmt.Errorf("%s expect 1 expression on left", errMsgPrefix(e.Pos)))
		}
		bin.Left = list[0]
	}

	left := bin.Left.getLeftValue(block, errs)
	right, es := bin.Right.checkSingleValueContextExpression(block)
	*errs = append(*errs, es...)
	if left == nil || right == nil {
		return
	}
	result := left.Clone()
	result.Pos = e.Pos
	if err := right.rightValueValid(); err != nil {
		*errs = append(*errs, err)
		return result
	}
	if bin.Left.Type == ExpressionTypeIdentifier &&
		e.IsStatementExpression == false {
		/*
			var a = 1
			print(a += 1)
		*/
		t := bin.Left.Data.(*ExpressionIdentifier)
		if t.Variable != nil {
			t.Variable.Used = true
		}
	}
	convertExpressionToNeed(bin.Right, left, right)
	/*
		var  s string
		s += "11111111"
	*/
	if left.Type == VariableTypeString {
		if right.Type == VariableTypeString &&
			(e.Type == ExpressionTypePlusAssign) {
			return result
		}
	}
	//number
	if e.Type == ExpressionTypePlusAssign ||
		e.Type == ExpressionTypeMinusAssign ||
		e.Type == ExpressionTypeMulAssign ||
		e.Type == ExpressionTypeDivAssign ||
		e.Type == ExpressionTypeModAssign {
		if left.assignAble(errs, right) {
			return result
		}
		if left.IsInteger() && right.IsInteger() && bin.Right.IsLiteral() {
			bin.Right.ConvertToNumber(left.Type)
			return result
		}
		if left.IsFloat() && right.IsFloat() && bin.Right.IsLiteral() {
			bin.Right.ConvertToNumber(left.Type)
			return result
		}
	}
	if e.Type == ExpressionTypeAndAssign ||
		e.Type == ExpressionTypeOrAssign ||
		e.Type == ExpressionTypeXorAssign {
		if left.IsInteger() && left.assignAble(errs, right) {
			return result
		}
	}
	if e.Type == ExpressionTypeLshAssign ||
		e.Type == ExpressionTypeRshAssign {
		if left.IsInteger() && right.IsInteger() {
			if right.Type == VariableTypeLong {
				bin.Right.ConvertToNumber(VariableTypeInt)
			}
			return result
		}
	}
	*errs = append(*errs, fmt.Errorf("%s cannot apply algorithm '%s' on '%s' and '%s'",
		errMsgPrefix(e.Pos),
		e.Description,
		left.TypeString(),
		right.TypeString()))

	return result
}
