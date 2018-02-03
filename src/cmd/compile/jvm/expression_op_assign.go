package jvm

import (
	"github.com/756445638/lucy/src/cmd/compile/ast"
	"github.com/756445638/lucy/src/cmd/compile/jvm/cg"
)

func (m *MakeExpression) buildStrPlusAssi(class *cg.ClassHighLevel, code *cg.AttributeCode, e *ast.Expression, context *Context) (maxstack uint16) {
	bin := e.Data.(*ast.ExpressionBinary)
	maxstack, remainStack, op, _, classname, fieldname, fieldDescriptor := m.getLeftValue(class, code, bin.Left, context)
	code.Codes[code.CodeLength] = cg.OP_new
	class.InsertClasses("java/lang/StringBuilder", code.Codes[code.CodeLength+1:code.CodeLength+3])
	code.Codes[code.CodeLength+3] = cg.OP_dup
	code.CodeLength += 4
	code.Codes[code.CodeLength] = cg.OP_invokespecial
	class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
		Class:      "java/lang/StringBuilder",
		Name:       specail_method_init,
		Descriptor: "()V",
	}, code.Codes[code.CodeLength+1:code.CodeLength+3])
	code.CodeLength += 3
	if t := remainStack + 2; t > maxstack {
		maxstack = t
	}
	currentStack := remainStack + 1 //
	stack, _ := m.build(class, code, bin.Left, context)
	if t := currentStack + stack; t > maxstack {
		maxstack = t
	}
	//append origin string
	code.Codes[code.CodeLength] = cg.OP_invokevirtual
	class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
		Class:      "java/lang/StringBuilder",
		Name:       `append`,
		Descriptor: "(Ljava/lang/String;)Ljava/lang/StringBuilder;",
	}, code.Codes[code.CodeLength+1:code.CodeLength+3])
	code.CodeLength += 3
	stack, es := m.build(class, code, bin.Right, context)
	backPatchEs(es, code)
	if t := currentStack + stack; t > maxstack {
		maxstack = t
	}
	m.stackTop2String(class, code, bin.Right.VariableType) //conver to string
	//append right
	code.Codes[code.CodeLength] = cg.OP_invokevirtual
	class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
		Class:      "java/lang/StringBuilder",
		Name:       `append`,
		Descriptor: "(Ljava/lang/String;)Ljava/lang/StringBuilder;",
	}, code.Codes[code.CodeLength+1:code.CodeLength+3])
	code.CodeLength += 3

	// tostring
	code.Codes[code.CodeLength] = cg.OP_invokevirtual
	class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
		Class:      "java/lang/StringBuilder",
		Name:       `toString`,
		Descriptor: "()Ljava/lang/String;",
	}, code.Codes[code.CodeLength+1:code.CodeLength+3])
	code.CodeLength += 3

	if e.IsStatementExpression == false {
		currentStack += m.controlStack2FitAssign(code, op, bin.Left.VariableType)
	}
	//copy op
	for _, v := range op {
		code.Codes[code.CodeLength] = v
		code.CodeLength++
	}
	if classname != "" { // must be put field static or not
		class.InsertFieldRefConst(cg.CONSTANT_Fieldref_info_high_level{
			Class:      classname,
			Name:       fieldname,
			Descriptor: fieldDescriptor,
		}, code.Codes[code.CodeLength:code.CodeLength+2])
		code.CodeLength += 2
	}
	return

}
func (m *MakeExpression) buildOpAssign(class *cg.ClassHighLevel, code *cg.AttributeCode, e *ast.Expression, context *Context) (maxstack uint16) {
	bin := e.Data.(*ast.ExpressionBinary)
	if bin.Left.VariableType.Typ == ast.VARIABLE_TYPE_STRING {
		return m.buildStrPlusAssi(class, code, e, context)
	}
	maxstack, remainStack, op, _, classname, fieldname, fieldDescriptor := m.getLeftValue(class, code, bin.Left, context)
	//left value must can be used as right value,
	stack, _ := m.build(class, code, bin.Left, context) // load it`s value
	if t := stack + remainStack; t > maxstack {
		maxstack = t
	}
	currentStack := bin.Left.VariableType.JvmSlotSize() + remainStack
	if currentStack > maxstack {
		maxstack = currentStack
	}
	stack, _ = m.build(class, code, bin.Right, context)
	if t := currentStack + stack; t > maxstack {
		maxstack = t
	}
	//convert stack top to same type
	if bin.Left.VariableType.IsInteger() {
		if bin.Left.VariableType.JvmSlotSize() != bin.Right.VariableType.JvmSlotSize() {
			m.numberTypeConverter(code, bin.Right.VariableType.Typ, bin.Left.VariableType.Typ)
		} else if bin.Right.VariableType.IsFloat() {
			m.numberTypeConverter(code, bin.Right.VariableType.Typ, bin.Left.VariableType.Typ)
		}
	}
	currentStack += bin.Left.VariableType.JvmSlotSize()
	if currentStack > maxstack {
		maxstack = currentStack // incase int->double
	}
	switch bin.Left.VariableType.Typ {
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
		} else {
			panic("... ")
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
		} else {
			panic("... ")
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
		} else {
			panic("... ")
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
		} else {
			panic("... ")
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
		} else {
			panic("... ")
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
		} else {
			panic("... ")
		}
	}
	currentStack -= bin.Left.VariableType.JvmSlotSize() // stack reduce
	if e.IsStatementExpression == false {
		currentStack += m.controlStack2FitAssign(code, op, bin.Left.VariableType)
		if currentStack > maxstack {
			maxstack = currentStack
		}
	}
	//copy op
	for _, v := range op {
		code.Codes[code.CodeLength] = v
		code.CodeLength++
	}
	if classname != "" { // must be put field static or not
		class.InsertFieldRefConst(cg.CONSTANT_Fieldref_info_high_level{
			Class:      classname,
			Name:       fieldname,
			Descriptor: fieldDescriptor,
		}, code.Codes[code.CodeLength:code.CodeLength+2])
		code.CodeLength += 2
	}

	return
}