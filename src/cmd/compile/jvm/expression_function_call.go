package jvm

import (
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/ast"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/jvm/cg"
)

func (m *MakeExpression) buildFunctionCall(class *cg.ClassHighLevel, code *cg.AttributeCode,
	e *ast.Expression, context *Context, state *StackMapState) (maxstack uint16) {
	call := e.Data.(*ast.ExpressionFunctionCall)
	if call.Func.IsBuildin {
		return m.mkBuildinFunctionCall(class, code, e, context, state)
	}
	if call.Func.TemplateFunction != nil {
		return m.buildTemplateFunctionCall(class, code, e, context, state)
	}
	if call.Func.IsClosureFunction == false {
		maxstack = m.buildCallArgs(class, code, call.Args, call.Func.Typ.ParameterList, context, state)
		code.Codes[code.CodeLength] = cg.OP_invokestatic
		class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
			Class:      call.Func.ClassMethod.Class.Name,
			Method:     call.Func.ClassMethod.Name,
			Descriptor: call.Func.ClassMethod.Descriptor,
		}, code.Codes[code.CodeLength+1:code.CodeLength+3])
		code.CodeLength += 3
	} else {
		//closure function call
		//load object
		if context.function.Closure.ClosureFunctionExist(call.Func) {
			copyOP(code, loadSimpleVarOp(ast.VARIABLE_TYPE_OBJECT, 0)...)
			code.Codes[code.CodeLength] = cg.OP_getfield
			class.InsertFieldRefConst(cg.CONSTANT_Fieldref_info_high_level{
				Class:      class.Name,
				Field:      call.Func.Name,
				Descriptor: "L" + call.Func.ClassMethod.Class.Name + ";",
			}, code.Codes[code.CodeLength+1:code.CodeLength+3])
			code.CodeLength += 3
		} else {
			copyOP(code, loadSimpleVarOp(ast.VARIABLE_TYPE_OBJECT, call.Func.VarOffSet)...)
		}
		state.pushStack(class, state.newObjectVariableType(call.Func.ClassMethod.Class.Name))
		defer state.popStack(1)
		stack := m.buildCallArgs(class, code, call.Args, call.Func.Typ.ParameterList, context, state)
		if t := 1 + stack; t > maxstack {
			maxstack = t
		}
		code.Codes[code.CodeLength] = cg.OP_invokevirtual
		class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
			Class:      call.Func.ClassMethod.Class.Name,
			Method:     call.Func.Name,
			Descriptor: call.Func.ClassMethod.Descriptor,
		}, code.Codes[code.CodeLength+1:code.CodeLength+3])
		code.CodeLength += 3
	}

	if e.IsStatementExpression {
		if e.CallHasReturnValue() == false {
			// nothing to do
		} else if len(e.Values) == 1 {
			if 2 == jvmSize(e.Values[0]) {
				code.Codes[code.CodeLength] = cg.OP_pop2
			} else {
				code.Codes[code.CodeLength] = cg.OP_pop
			}
			code.CodeLength++
		} else { // > 1
			code.Codes[code.CodeLength] = cg.OP_pop // arraylist object on stack
			code.CodeLength++
		}
	}

	if e.CallHasReturnValue() == false { // nothing
	} else if len(e.Values) == 1 {
		if t := jvmSize(e.Values[0]); t > maxstack {
			maxstack = t
		}
	} else { // > 1
		if 1 > maxstack {
			maxstack = 1
		}
	}
	return
}
