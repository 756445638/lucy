package jvm

import (
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/ast"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/jvm/cg"
)

func (buildExpression *BuildExpression) buildStringSlice(class *cg.ClassHighLevel, code *cg.AttributeCode,
	e *ast.Expression, context *Context, state *StackMapState) (maxStack uint16) {
	slice := e.Data.(*ast.ExpressionSlice)
	maxStack, _ = buildExpression.build(class, code, slice.Expression, context, state)
	state.pushStack(class, state.newObjectVariableType(javaStringClass))
	// build start
	stack, _ := buildExpression.build(class, code, slice.Start, context, state)
	if t := 1 + stack; t > maxStack {
		maxStack = t
	}
	state.pushStack(class, slice.Start.ExpressionValue)
	stack, _ = buildExpression.build(class, code, slice.End, context, state)
	if t := 2 + stack; t > maxStack {
		maxStack = t
	}
	code.Codes[code.CodeLength] = cg.OP_invokevirtual
	class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
		Class:      javaStringClass,
		Method:     "substring",
		Descriptor: "(II)Ljava/lang/String;",
	}, code.Codes[code.CodeLength+1:code.CodeLength+3])
	code.CodeLength += 3
	return
}

func (buildExpression *BuildExpression) buildSlice(class *cg.ClassHighLevel, code *cg.AttributeCode,
	e *ast.Expression, context *Context, state *StackMapState) (maxStack uint16) {
	stackLength := len(state.Stacks)
	defer func() {
		state.popStack(len(state.Stacks) - stackLength)
	}()
	slice := e.Data.(*ast.ExpressionSlice)
	if slice.Expression.ExpressionValue.Type == ast.VariableTypeString {
		return buildExpression.buildStringSlice(class, code, e, context, state)
	}
	meta := ArrayMetas[slice.Expression.ExpressionValue.Array.Type]
	maxStack, _ = buildExpression.build(class, code, slice.Expression, context, state)
	state.pushStack(class, slice.Expression.ExpressionValue)
	// build start
	stack, _ := buildExpression.build(class, code, slice.Start, context, state)
	if t := 1 + stack; t > maxStack {
		maxStack = t
	}
	state.pushStack(class, slice.Start.ExpressionValue)
	stack, _ = buildExpression.build(class, code, slice.End, context, state)
	if t := 2 + stack; t > maxStack {
		maxStack = t
	}
	code.Codes[code.CodeLength] = cg.OP_invokevirtual
	class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
		Class:      meta.className,
		Method:     "slice",
		Descriptor: meta.sliceDescriptor,
	}, code.Codes[code.CodeLength+1:code.CodeLength+3])
	code.CodeLength += 3
	return
}
