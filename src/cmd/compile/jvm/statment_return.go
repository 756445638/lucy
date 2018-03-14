package jvm

import (
	"github.com/756445638/lucy/src/cmd/compile/ast"
	"github.com/756445638/lucy/src/cmd/compile/jvm/cg"
)

func (m *MakeClass) buildReturnStatement(class *cg.ClassHighLevel, code *cg.AttributeCode, s *ast.StatementReturn, context *Context) (maxstack uint16) {
	if len(s.Function.Typ.ReturnList) == 0 {
		if context.Defers != nil && len(context.Defers) > 0 {
			code.Codes[code.CodeLength] = cg.OP_aconst_null
			code.CodeLength++
			if 1 > code.MaxStack {
				code.MaxStack = 1
			}
			context.endPcForException = code.CodeLength
			m.buildDefers(class, code, context, context.Defers, true)
		}
		code.Codes[code.CodeLength] = cg.OP_return
		code.CodeLength++
		return
	}
	if len(s.Function.Typ.ReturnList) == 1 {
		var es []*cg.JumpBackPatch
		maxstack, es = m.MakeExpression.build(class, code, s.Expressions[0], context)
		backPatchEs(es, code.CodeLength)
		if s.Expressions[0].VariableType.IsNumber() && s.Expressions[0].VariableType.Typ != s.Function.Typ.ReturnList[0].Typ.Typ {
			m.MakeExpression.numberTypeConverter(code, s.Expressions[0].VariableType.Typ, s.Function.Typ.ReturnList[0].Typ.Typ)
		}
		// execute defer first
		if context.Defers != nil && len(context.Defers) > 0 {
			//return value  is on stack,  store it temp var
			copyOP(code,
				storeSimpleVarOp(s.Function.Typ.ReturnList[0].Typ.Typ, context.function.AutoVarForReturnBecauseOfDefer.Offset)...)
			code.Codes[code.CodeLength] = cg.OP_aconst_null
			code.CodeLength++
			if 1 > code.MaxStack {
				code.MaxStack = 1
			}
			context.endPcForException = code.CodeLength
			m.buildDefers(class, code, context, context.Defers, true)
			code.MaxStack += s.Function.Typ.ReturnList[0].Typ.JvmSlotSize()
			//restore the stack
			copyOP(code,
				loadSimpleVarOp(s.Function.Typ.ReturnList[0].Typ.Typ, context.function.AutoVarForReturnBecauseOfDefer.Offset)...)
		}
		switch s.Function.Typ.ReturnList[0].Typ.Typ {
		case ast.VARIABLE_TYPE_BOOL:
			fallthrough
		case ast.VARIABLE_TYPE_BYTE:
			fallthrough
		case ast.VARIABLE_TYPE_SHORT:
			fallthrough
		case ast.VARIABLE_TYPE_INT:
			code.Codes[code.CodeLength] = cg.OP_ireturn
		case ast.VARIABLE_TYPE_LONG:
			code.Codes[code.CodeLength] = cg.OP_lreturn
		case ast.VARIABLE_TYPE_FLOAT:
			code.Codes[code.CodeLength] = cg.OP_freturn
		case ast.VARIABLE_TYPE_DOUBLE:
			code.Codes[code.CodeLength] = cg.OP_dreturn
		case ast.VARIABLE_TYPE_STRING:
			fallthrough
		case ast.VARIABLE_TYPE_OBJECT:
			fallthrough
		case ast.VARIABLE_TYPE_MAP:
			fallthrough
		case ast.VARIABLE_TYPE_ARRAY:
			code.Codes[code.CodeLength] = cg.OP_areturn
		default:
			panic("......a")
		}
		code.CodeLength++
		return
	}
	//new a array list
	code.Codes[code.CodeLength] = cg.OP_new
	class.InsertClassConst(java_arrylist_class, code.Codes[code.CodeLength+1:code.CodeLength+3])
	code.Codes[code.CodeLength+3] = cg.OP_dup // dup on stack
	code.CodeLength += 4
	//call init
	code.Codes[code.CodeLength] = cg.OP_invokespecial
	class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
		Class:      java_arrylist_class,
		Name:       special_method_init,
		Descriptor: "()V",
	}, code.Codes[code.CodeLength+1:code.CodeLength+3])
	code.CodeLength += 3
	maxstack = 2 // max stack is 2
	currentStack := uint16(1)
	for _, v := range s.Expressions {
		code.Codes[code.CodeLength] = cg.OP_dup // dup array list
		code.CodeLength++
		currentStack++
		if currentStack > maxstack {
			maxstack = maxstack
		}
		if v.IsCall() && len(v.VariableTypes) > 1 {
			if currentStack > maxstack {
				maxstack = maxstack
			} // make the call
			stack, _ := m.MakeExpression.build(class, code, v, context)
			if t := currentStack + stack; t > maxstack {
				maxstack = t
			}
			code.Codes[code.CodeLength] = cg.OP_invokevirtual
			class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
				Class:      java_arrylist_class,
				Name:       "addAll",
				Descriptor: "(Ljava/util/Collection;)Z",
			}, code.Codes[code.CodeLength+1:code.CodeLength+3])
			code.Codes[code.CodeLength+3] = cg.OP_pop
			code.CodeLength += 4
			continue
		}
		variableType := v.VariableType
		if v.IsCall() {
			variableType = v.VariableTypes[0]
		}
		stack, es := m.MakeExpression.build(class, code, v, context)
		backPatchEs(es, code.CodeLength)
		if t := stack + currentStack; t > maxstack {
			maxstack = t
		}
		//convert to object
		PrimitiveObjectConverter.putPrimitiveInObjectStaticWay(class, code, variableType)
		// append
		code.Codes[code.CodeLength] = cg.OP_invokevirtual
		class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
			Class:      java_arrylist_class,
			Name:       "add",
			Descriptor: "(Ljava/lang/Object;)Z",
		}, code.Codes[code.CodeLength+1:code.CodeLength+3])
		code.Codes[code.CodeLength+3] = cg.OP_pop
		code.CodeLength += 4
		currentStack = 1
	}
	if context.Defers != nil && len(context.Defers) > 0 {
		//return value  is on stack,  store it temp var
		copyOP(code,
			storeSimpleVarOp(ast.VARIABLE_TYPE_OBJECT, context.function.AutoVarForReturnBecauseOfDefer.Offset)...)
		code.Codes[code.CodeLength] = cg.OP_aconst_null
		code.CodeLength++
		if 1 > code.MaxStack {
			code.MaxStack = 1
		}
		context.endPcForException = code.CodeLength
		m.buildDefers(class, code, context, context.Defers, true)
		code.MaxStack += s.Function.Typ.ReturnList[0].Typ.JvmSlotSize()
		//restore the stack
		copyOP(code,
			loadSimpleVarOp(ast.VARIABLE_TYPE_OBJECT, context.function.AutoVarForReturnBecauseOfDefer.Offset)...)
	}
	code.Codes[code.CodeLength] = cg.OP_areturn
	code.CodeLength++
	return
}
