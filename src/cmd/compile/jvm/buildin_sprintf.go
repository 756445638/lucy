package jvm

import (
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/ast"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/jvm/cg"
)

func (m *MakeExpression) mkBuildinSprintf(class *cg.ClassHighLevel, code *cg.AttributeCode, e *ast.Expression,
	context *Context, state *StackMapState) (maxstack uint16) {
	length := len(state.Stacks)
	defer func() {
		state.popStack(len(state.Stacks) - length)
	}()
	// format,must be string
	call := e.Data.(*ast.ExpressionFunctionCall)
	meta := call.Meta.(*ast.BuildinFunctionSprintfMeta)
	maxstack, _ = m.build(class, code, meta.Format, context, state)
	loadInt32(class, code, int32(meta.ArgsLength))
	code.Codes[code.CodeLength] = cg.OP_anewarray
	class.InsertClassConst("java/lang/Object", code.Codes[code.CodeLength+1:code.CodeLength+3])
	code.CodeLength += 3
	currentStack := uint16(2)
	if currentStack > maxstack {
		maxstack = currentStack
	}
	state.pushStack(class, state.newObjectVariableType(java_string_class))
	objectArray := &ast.VariableType{}
	objectArray.ArrayType = state.newObjectVariableType(java_root_class)
	state.pushStack(class, objectArray)
	index := int32(0)
	for _, v := range call.Args {
		if v.MayHaveMultiValue() && len(v.Values) > 1 {
			currentStack = 2
			stack, _ := m.build(class, code, v, context, state)
			if t := currentStack + stack; t > maxstack {
				maxstack = t
			}
			// store in temp var
			arrayListPacker.buildStoreArrayListAutoVar(code, context)
			for kk, _ := range v.Values {
				currentStack = 2
				code.Codes[code.CodeLength] = cg.OP_dup
				code.CodeLength++
				loadInt32(class, code, index)
				currentStack += 2
				stack = arrayListPacker.unPackObject(class, code, kk, context)
				if t := currentStack + stack; t > maxstack {
					maxstack = t
				}
				code.Codes[code.CodeLength] = cg.OP_aastore
				code.CodeLength++
				index++
			}
			continue
		}
		currentStack = 2
		code.Codes[code.CodeLength] = cg.OP_dup
		code.CodeLength++
		loadInt32(class, code, index)
		currentStack += 2
		stack, es := m.build(class, code, v, context, state)
		if len(es) > 0 {
			backPatchEs(es, code.CodeLength)
			state.pushStack(class, objectArray)
			state.pushStack(class, &ast.VariableType{Typ: ast.VARIABLE_TYPE_INT})
			state.pushStack(class, v.Value)
			context.MakeStackMap(code, state, code.CodeLength)
			state.popStack(3) // bool value
		}
		if t := currentStack + stack; t > maxstack {
			maxstack = t
		}
		if v.Value.IsPointer() == false {
			typeConverter.putPrimitiveInObject(class, code, v.Value)
		}
		code.Codes[code.CodeLength] = cg.OP_aastore
		code.CodeLength++
		index++
	}
	code.Codes[code.CodeLength] = cg.OP_invokestatic
	class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
		Class:      java_string_class,
		Method:     "format",
		Descriptor: "(Ljava/lang/String;[Ljava/lang/Object;)Ljava/lang/String;",
	},
		code.Codes[code.CodeLength+1:code.CodeLength+3])
	code.CodeLength += 3
	if e.IsStatementExpression {
		code.Codes[code.CodeLength] = cg.OP_pop
		code.CodeLength++
	}
	return
}