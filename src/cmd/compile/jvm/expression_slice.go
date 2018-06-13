package jvm

import (
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/ast"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/jvm/cg"
)

func (m *MakeExpression) buildSlice(class *cg.ClassHighLevel, code *cg.AttributeCode,
	e *ast.Expression, context *Context, state *StackMapState) (maxStack uint16) {
	stackLength := len(state.Stacks)
	defer func() {
		state.popStack(len(state.Stacks) - stackLength)
	}()
	sliceOn := e.Data.(*ast.ExpressionSlice)
	meta := ArrayMetas[sliceOn.Array.Value.ArrayType.Typ]
	maxStack, _ = m.build(class, code, sliceOn.Array, context, state)
	state.pushStack(class, sliceOn.Array.Value)
	// build start
	stack, _ := m.build(class, code, sliceOn.Start, context, state)
	if t := 1 + stack; t > maxStack {
		maxStack = t
	}
	state.pushStack(class, sliceOn.Start.Value)
	stack, _ = m.build(class, code, sliceOn.End, context, state)
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
