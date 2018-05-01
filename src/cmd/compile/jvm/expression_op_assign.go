package jvm

import (
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/ast"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/jvm/cg"
)

/*
	s := "123";
	s += "456";
*/
func (m *MakeExpression) buildStrPlusAssign(class *cg.ClassHighLevel, code *cg.AttributeCode, e *ast.Expression, context *Context, state *StackMapState) (maxstack uint16) {
	bin := e.Data.(*ast.ExpressionBinary)
	stackLength := len(state.Stacks)
	defer func() {
		state.popStack(len(state.Stacks) - stackLength)
	}()
	maxstack, remainStack, op, _, classname, name, descriptor := m.getLeftValue(class, code, bin.Left, context, state)
	code.Codes[code.CodeLength] = cg.OP_new
	class.InsertClassConst("java/lang/StringBuilder", code.Codes[code.CodeLength+1:code.CodeLength+3])
	code.Codes[code.CodeLength+3] = cg.OP_dup
	code.CodeLength += 4
	code.Codes[code.CodeLength] = cg.OP_invokespecial
	class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
		Class:      "java/lang/StringBuilder",
		Method:     special_method_init,
		Descriptor: "()V",
	}, code.Codes[code.CodeLength+1:code.CodeLength+3])
	code.CodeLength += 3
	if t := remainStack + 2; t > maxstack {
		maxstack = t
	}
	currentStack := remainStack + 1 //
	stack, _ := m.build(class, code, bin.Left, context, state)
	if t := currentStack + stack; t > maxstack {
		maxstack = t
	}
	m.stackTop2String(class, code, bin.Right.Value, context, state) //conver to string
	//append origin string
	code.Codes[code.CodeLength] = cg.OP_invokevirtual
	class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
		Class:      "java/lang/StringBuilder",
		Method:     `append`,
		Descriptor: "(Ljava/lang/String;)Ljava/lang/StringBuilder;",
	}, code.Codes[code.CodeLength+1:code.CodeLength+3])
	code.CodeLength += 3
	stack, _ = m.build(class, code, bin.Right, context, state)
	if t := currentStack + stack; t > maxstack {
		maxstack = t
	}
	m.stackTop2String(class, code, bin.Right.Value, context, state) //conver to string
	if bin.Right.Value.IsPointer() && bin.Right.Value.Typ != ast.VARIABLE_TYPE_STRING {
		if t := 2 + currentStack; t > maxstack {
			maxstack = t
		}
	}
	//append right
	code.Codes[code.CodeLength] = cg.OP_invokevirtual
	class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
		Class:      "java/lang/StringBuilder",
		Method:     `append`,
		Descriptor: "(Ljava/lang/String;)Ljava/lang/StringBuilder;",
	}, code.Codes[code.CodeLength+1:code.CodeLength+3])
	code.CodeLength += 3
	// tostring
	code.Codes[code.CodeLength] = cg.OP_invokevirtual
	class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
		Class:      "java/lang/StringBuilder",
		Method:     `toString`,
		Descriptor: "()Ljava/lang/String;",
	}, code.Codes[code.CodeLength+1:code.CodeLength+3])
	code.CodeLength += 3
	if e.IsStatementExpression == false {
		currentStack += m.controlStack2FitAssign(code, op, classname, bin.Left.Value)
	}
	//copy op
	copyOPLeftValue(class, code, op, classname, name, descriptor)
	return

}
func (m *MakeExpression) buildOpAssign(class *cg.ClassHighLevel, code *cg.AttributeCode, e *ast.Expression, context *Context, state *StackMapState) (maxstack uint16) {
	bin := e.Data.(*ast.ExpressionBinary)
	if bin.Left.Value.Typ == ast.VARIABLE_TYPE_STRING {
		return m.buildStrPlusAssign(class, code, e, context, state)
	}
	maxstack, remainStack, op, _, classname, name, descriptor := m.getLeftValue(class, code, bin.Left, context, state)
	//left value must can be used as right value,
	stack, _ := m.build(class, code, bin.Left, context, state) // load it`s value
	if t := stack + remainStack; t > maxstack {
		maxstack = t
	}
	currentStack := jvmSize(bin.Left.Value) + remainStack // incase int -> long
	if currentStack > maxstack {
		maxstack = currentStack
	}
	stack, _ = m.build(class, code, bin.Right, context, state)
	if t := currentStack + stack; t > maxstack {
		maxstack = t
	}
	//convert stack top to same type
	if bin.Left.Value.Typ != bin.Right.Value.Typ {
		m.numberTypeConverter(code, bin.Right.Value.Typ, bin.Left.Value.Typ)
	}
	currentStack += jvmSize(bin.Left.Value)
	if currentStack > maxstack {
		maxstack = currentStack // incase int->double
	}
	switch bin.Left.Value.Typ {
	case ast.VARIABLE_TYPE_BYTE:
		if e.Typ == ast.EXPRESSION_TYPE_PLUS_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_iadd
			code.Codes[code.CodeLength+1] = cg.OP_i2b
			code.CodeLength += 2
		} else if e.Typ == ast.EXPRESSION_TYPE_MINUS_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_isub
			code.Codes[code.CodeLength+1] = cg.OP_i2b
			code.CodeLength += 2
		} else if e.Typ == ast.EXPRESSION_TYPE_MUL_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_imul
			code.Codes[code.CodeLength+1] = cg.OP_i2b
			code.CodeLength += 2
		} else if e.Typ == ast.EXPRESSION_TYPE_DIV_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_idiv
			code.Codes[code.CodeLength+1] = cg.OP_i2b
			code.CodeLength += 2
		} else if e.Typ == ast.EXPRESSION_TYPE_MOD_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_irem
			code.Codes[code.CodeLength+1] = cg.OP_i2b
			code.CodeLength += 2
		}
	case ast.VARIABLE_TYPE_SHORT:
		if e.Typ == ast.EXPRESSION_TYPE_PLUS_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_iadd
			code.Codes[code.CodeLength+1] = cg.OP_i2s
			code.CodeLength += 2
		} else if e.Typ == ast.EXPRESSION_TYPE_MINUS_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_isub
			code.Codes[code.CodeLength+1] = cg.OP_i2s
			code.CodeLength += 2
		} else if e.Typ == ast.EXPRESSION_TYPE_MUL_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_imul
			code.Codes[code.CodeLength+1] = cg.OP_i2s
			code.CodeLength += 2
		} else if e.Typ == ast.EXPRESSION_TYPE_DIV_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_idiv
			code.Codes[code.CodeLength+1] = cg.OP_i2s
			code.CodeLength += 2
		} else if e.Typ == ast.EXPRESSION_TYPE_MOD_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_irem
			code.Codes[code.CodeLength+1] = cg.OP_i2s
			code.CodeLength += 2
		}

	case ast.VARIABLE_TYPE_INT:
		if e.Typ == ast.EXPRESSION_TYPE_PLUS_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_iadd
			code.CodeLength++
		} else if e.Typ == ast.EXPRESSION_TYPE_MINUS_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_isub
			code.CodeLength++
		} else if e.Typ == ast.EXPRESSION_TYPE_MUL_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_imul
			code.CodeLength++
		} else if e.Typ == ast.EXPRESSION_TYPE_DIV_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_idiv
			code.CodeLength++
		} else if e.Typ == ast.EXPRESSION_TYPE_MOD_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_irem
			code.CodeLength++
		}
	case ast.VARIABLE_TYPE_LONG:
		if e.Typ == ast.EXPRESSION_TYPE_PLUS_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_ladd
			code.CodeLength++
		} else if e.Typ == ast.EXPRESSION_TYPE_MINUS_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_lsub
			code.CodeLength++
		} else if e.Typ == ast.EXPRESSION_TYPE_MUL_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_lmul
			code.CodeLength++
		} else if e.Typ == ast.EXPRESSION_TYPE_DIV_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_ldiv
			code.CodeLength++
		} else if e.Typ == ast.EXPRESSION_TYPE_MOD_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_frem
			code.CodeLength++
		}
	case ast.VARIABLE_TYPE_FLOAT:
		if e.Typ == ast.EXPRESSION_TYPE_PLUS_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_ladd
			code.CodeLength++
		} else if e.Typ == ast.EXPRESSION_TYPE_MINUS_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_lsub
			code.CodeLength++
		} else if e.Typ == ast.EXPRESSION_TYPE_MUL_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_lmul
			code.CodeLength++
		} else if e.Typ == ast.EXPRESSION_TYPE_DIV_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_ldiv
			code.CodeLength++
		} else if e.Typ == ast.EXPRESSION_TYPE_MOD_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_frem
			code.CodeLength++
		}
	case ast.VARIABLE_TYPE_DOUBLE:
		if e.Typ == ast.EXPRESSION_TYPE_PLUS_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_dadd
			code.CodeLength++
		} else if e.Typ == ast.EXPRESSION_TYPE_MINUS_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_dsub
			code.CodeLength++
		} else if e.Typ == ast.EXPRESSION_TYPE_MUL_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_dmul
			code.CodeLength++
		} else if e.Typ == ast.EXPRESSION_TYPE_DIV_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_ddiv
			code.CodeLength++
		} else if e.Typ == ast.EXPRESSION_TYPE_MOD_ASSIGN {
			code.Codes[code.CodeLength] = cg.OP_drem
			code.CodeLength++
		}
	}
	if classname == java_hashmap_class && e.Value.IsPointer() == false { // map destination
		typeConverter.putPrimitiveInObject(class, code, e.Value)
	}
	currentStack -= jvmSize(bin.Left.Value) // stack reduce
	if e.IsStatementExpression == false {
		currentStack += m.controlStack2FitAssign(code, op, classname, bin.Left.Value)
		if currentStack > maxstack {
			maxstack = currentStack
		}
	}
	//copy op
	copyOPLeftValue(class, code, op, classname, name, descriptor)
	if classname == java_hashmap_class && e.Value.IsPointer() == false { // map destination
		typeConverter.getFromObject(class, code, e.Value)
	}
	return
}
