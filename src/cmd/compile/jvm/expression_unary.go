package jvm

import (
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/ast"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/jvm/cg"
)

func (buildExpression *BuildExpression) buildUnary(
	class *cg.ClassHighLevel,
	code *cg.AttributeCode,
	e *ast.Expression,
	context *Context,
	state *StackMapState) (maxStack uint16) {

	if e.Type == ast.ExpressionTypeNegative {
		ee := e.Data.(*ast.Expression)
		maxStack = buildExpression.build(class, code, ee, context, state)
		switch e.Value.Type {
		case ast.VariableTypeByte:
			fallthrough
		case ast.VariableTypeShort:
			fallthrough
		case ast.VariableTypeChar:
			fallthrough
		case ast.VariableTypeInt:
			code.Codes[code.CodeLength] = cg.OP_ineg
		case ast.VariableTypeFloat:
			code.Codes[code.CodeLength] = cg.OP_fneg
		case ast.VariableTypeDouble:
			code.Codes[code.CodeLength] = cg.OP_dneg
		case ast.VariableTypeLong:
			code.Codes[code.CodeLength] = cg.OP_lneg
		}
		code.CodeLength++
		return
	}
	if e.Type == ast.ExpressionTypeBitwiseNot {
		ee := e.Data.(*ast.Expression)
		maxStack = buildExpression.build(class, code, ee, context, state)
		if t := jvmSlotSize(ee.Value) * 2; t > maxStack {
			maxStack = t
		}
		switch e.Value.Type {
		case ast.VariableTypeByte:
			code.Codes[code.CodeLength] = cg.OP_bipush
			code.Codes[code.CodeLength+1] = 255
			code.Codes[code.CodeLength+2] = cg.OP_ixor
			code.CodeLength += 3
			if 2 > maxStack {
				maxStack = 2
			}
		case ast.VariableTypeShort:
			fallthrough
		case ast.VariableTypeChar:
			code.Codes[code.CodeLength] = cg.OP_sipush
			code.Codes[code.CodeLength+1] = 255
			code.Codes[code.CodeLength+2] = 255
			code.Codes[code.CodeLength+3] = cg.OP_ixor
			code.CodeLength += 4
			if 2 > maxStack {
				maxStack = 2
			}
		case ast.VariableTypeInt:
			code.Codes[code.CodeLength] = cg.OP_ldc_w
			class.InsertIntConst(-1, code.Codes[code.CodeLength+1:code.CodeLength+3])
			code.Codes[code.CodeLength+3] = cg.OP_ixor
			code.CodeLength += 4
			if 2 > maxStack {
				maxStack = 2
			}
		case ast.VariableTypeLong:
			code.Codes[code.CodeLength] = cg.OP_ldc2_w
			class.InsertLongConst(-1, code.Codes[code.CodeLength+1:code.CodeLength+3])
			code.Codes[code.CodeLength+3] = cg.OP_lxor
			code.CodeLength += 4
			if 4 > maxStack {
				maxStack = 4
			}
		}
		return
	}
	if e.Type == ast.ExpressionTypeNot {
		ee := e.Data.(*ast.Expression)
		maxStack = buildExpression.build(class, code, ee, context, state)
		exit := (&cg.Exit{}).Init(cg.OP_ifeq, code)
		code.Codes[code.CodeLength] = cg.OP_iconst_0
		code.CodeLength++
		exit2 := (&cg.Exit{}).Init(cg.OP_goto, code)
		context.MakeStackMap(code, state, code.CodeLength)
		writeExits([]*cg.Exit{exit}, code.CodeLength)
		code.Codes[code.CodeLength] = cg.OP_iconst_1
		code.CodeLength++
		state.pushStack(class, ee.Value)
		defer state.popStack(1)
		writeExits([]*cg.Exit{exit2}, code.CodeLength)
		context.MakeStackMap(code, state, code.CodeLength)
	}
	return
}
