package jvm

import (
	"fmt"

	"github.com/756445638/lucy/src/cmd/compile/ast"
	"github.com/756445638/lucy/src/cmd/compile/jvm/cg"
)

type MakeExpression struct {
	MakeClass *MakeClass
}

func (m *MakeExpression) build(class *cg.ClassHighLevel, code *cg.AttributeCode, e *ast.Expression, context *Context) (maxstack uint16, exits []*cg.JumpBackPatch) {
	fmt.Println(e.OpName())
	if e.IsCompileAutoExpression == false {
		context.appendLimeNumberAndSourceFile(e.Pos, code, class)
	}
	switch e.Typ {
	case ast.EXPRESSION_TYPE_NULL:
		code.Codes[code.CodeLength] = cg.OP_aconst_null
		code.CodeLength++
		maxstack = 1
	case ast.EXPRESSION_TYPE_BOOL:
		if e.Data.(bool) {
			code.Codes[code.CodeLength] = cg.OP_iconst_1
		} else {
			code.Codes[code.CodeLength] = cg.OP_iconst_0
		}
		code.CodeLength++
		maxstack = 1
	case ast.EXPRESSION_TYPE_BYTE:
		e.Data = int32(e.Data.(byte))
		fallthrough
	case ast.EXPRESSION_TYPE_INT:
		loadInt32(class, code, e.Data.(int32))
		maxstack = 1
	case ast.EXPRESSION_TYPE_LONG:
		if e.Data.(int64) == 0 {
			code.Codes[code.CodeLength] = cg.OP_lconst_0
			code.CodeLength++
		} else if e.Data.(int64) == 1 {
			code.Codes[code.CodeLength] = cg.OP_lconst_1
			code.CodeLength++
		} else {
			code.Codes[code.CodeLength] = cg.OP_ldc2_w
			class.InsertLongConst(e.Data.(int64), code.Codes[code.CodeLength+1:code.CodeLength+3])
			code.CodeLength += 3
		}
		maxstack = 2
	case ast.EXPRESSION_TYPE_FLOAT:
		if e.Data.(float32) == 0.0 {
			code.Codes[code.CodeLength] = cg.OP_fconst_0
			code.CodeLength++
		} else if e.Data.(float32) == 1.0 {
			code.Codes[code.CodeLength] = cg.OP_fconst_1
			code.CodeLength++
		} else if e.Data.(float32) == 2.0 {
			code.Codes[code.CodeLength] = cg.OP_fconst_2
			code.CodeLength++
		} else {
			code.Codes[code.CodeLength] = cg.OP_ldc_w
			class.InsertFloatConst(e.Data.(float32), code.Codes[code.CodeLength+1:code.CodeLength+3])
			code.CodeLength += 3
		}
		maxstack = 1
	case ast.EXPRESSION_TYPE_DOUBLE:
		if e.Data.(float64) == 0.0 {
			code.Codes[code.CodeLength] = cg.OP_dconst_0
			code.CodeLength++
		} else if e.Data.(float64) == 1.0 {
			code.Codes[code.CodeLength] = cg.OP_dconst_1
			code.CodeLength++
		} else {
			code.Codes[code.CodeLength] = cg.OP_ldc2_w
			class.InsertDoubleConst(e.Data.(float64), code.Codes[code.CodeLength+1:code.CodeLength+3])
			code.CodeLength += 3
		}
		maxstack = 2
	case ast.EXPRESSION_TYPE_STRING:
		code.Codes[code.CodeLength] = cg.OP_ldc_w
		class.InsertStringConst(e.Data.(string), code.Codes[code.CodeLength+1:code.CodeLength+3])
		code.CodeLength += 3
		maxstack = 1
	//binary expression
	case ast.EXPRESSION_TYPE_LOGICAL_OR:
		fallthrough
	case ast.EXPRESSION_TYPE_LOGICAL_AND:
		maxstack, exits = m.buildLogical(class, code, e, context)
	case ast.EXPRESSION_TYPE_OR:
		fallthrough
	case ast.EXPRESSION_TYPE_AND:
		fallthrough
	case ast.EXPRESSION_TYPE_LEFT_SHIFT:
		fallthrough
	case ast.EXPRESSION_TYPE_RIGHT_SHIFT:
		fallthrough
	case ast.EXPRESSION_TYPE_ADD:
		fallthrough
	case ast.EXPRESSION_TYPE_SUB:
		fallthrough
	case ast.EXPRESSION_TYPE_MUL:
		fallthrough
	case ast.EXPRESSION_TYPE_DIV:
		fallthrough
	case ast.EXPRESSION_TYPE_MOD:
		maxstack = m.buildArithmetic(class, code, e, context)
	//
	case ast.EXPRESSION_TYPE_ASSIGN:
		maxstack = m.buildAssign(class, code, e, context)
	case ast.EXPRESSION_TYPE_COLON_ASSIGN:
		maxstack = m.buildColonAssign(class, code, e, context)
	//
	case ast.EXPRESSION_TYPE_PLUS_ASSIGN:
		fallthrough
	case ast.EXPRESSION_TYPE_MINUS_ASSIGN:
		fallthrough
	case ast.EXPRESSION_TYPE_MUL_ASSIGN:
		fallthrough
	case ast.EXPRESSION_TYPE_DIV_ASSIGN:
		fallthrough
	case ast.EXPRESSION_TYPE_MOD_ASSIGN:
		maxstack = m.buildOpAssign(class, code, e, context)
	//
	case ast.EXPRESSION_TYPE_EQ:
		fallthrough
	case ast.EXPRESSION_TYPE_NE:
		fallthrough
	case ast.EXPRESSION_TYPE_GE:
		fallthrough
	case ast.EXPRESSION_TYPE_GT:
		fallthrough
	case ast.EXPRESSION_TYPE_LE:
		fallthrough
	case ast.EXPRESSION_TYPE_LT:
		maxstack = m.buildRelations(class, code, e, context)
	//
	case ast.EXPRESSION_TYPE_INDEX:
		maxstack = m.buildIndex(class, code, e, context)
	case ast.EXPRESSION_TYPE_DOT:
		maxstack = m.buildDot(class, code, e, context)

	//
	case ast.EXPRESSION_TYPE_METHOD_CALL:
		maxstack = m.buildMethodCall(class, code, e, context)
	case ast.EXPRESSION_TYPE_FUNCTION_CALL:
		maxstack = m.buildFunctionCall(class, code, e, context)
	//
	case ast.EXPRESSION_TYPE_INCREMENT:
		fallthrough
	case ast.EXPRESSION_TYPE_DECREMENT:
		fallthrough
	case ast.EXPRESSION_TYPE_PRE_INCREMENT:
		fallthrough
	case ast.EXPRESSION_TYPE_PRE_DECREMENT:
		maxstack = m.buildSelfIncrement(class, code, e, context)
	//
	case ast.EXPRESSION_TYPE_NEGATIVE:
		fallthrough
	case ast.EXPRESSION_TYPE_NOT:
		maxstack = m.buildUnary(class, code, e, context)
	//
	case ast.EXPRESSION_TYPE_IDENTIFIER:
		maxstack = m.buildIdentifer(class, code, e, context)
	case ast.EXPRESSION_TYPE_NEW:
		maxstack = m.buildNew(class, code, e, context)
	case ast.EXPRESSION_TYPE_FUNCTION:
	case ast.EXPRESSION_TYPE_CONVERTION_TYPE: // []byte(str)
		maxstack = m.buildTypeConvertion(class, code, e, context)
	case ast.EXPRESSION_TYPE_CONST: // const will analyse at ast stage
	case ast.EXPRESSSION_TYPE_SLICE:
		maxstack = m.buildSlice(class, code, e, context)
	case ast.EXPRESSION_TYPE_ARRAY:
		maxstack = m.buildArray(class, code, e, context)
	default:
		panic(e.OpName())
	}

	return
}

/*
	stack is 1
*/
func (m *MakeExpression) buildLoadArrayListAutoVar(code *cg.AttributeCode, context *Context) {
	switch context.function.AutoVarForMultiReturn.Offset {
	case 0:
		code.Codes[code.CodeLength] = cg.OP_aload_0
		code.CodeLength++
	case 1:
		code.Codes[code.CodeLength] = cg.OP_aload_1
		code.CodeLength++
	case 2:
		code.Codes[code.CodeLength] = cg.OP_aload_2
		code.CodeLength++
	case 3:
		code.Codes[code.CodeLength] = cg.OP_aload_3
		code.CodeLength++
	default:
		if context.function.AutoVarForMultiReturn.Offset > 255 {
			panic("local var offset over 255")
		}
		code.Codes[code.CodeLength] = cg.OP_aload
		code.Codes[code.CodeLength+1] = byte(context.function.AutoVarForMultiReturn.Offset)
		code.CodeLength += 2
	}
}

/*
	stack is 1
*/
func (m *MakeExpression) buildStoreArrayListAutoVar(code *cg.AttributeCode, context *Context) {
	switch context.function.AutoVarForMultiReturn.Offset {
	case 0:
		code.Codes[code.CodeLength] = cg.OP_astore_0
		code.CodeLength++
	case 1:
		code.Codes[code.CodeLength] = cg.OP_astore_1
		code.CodeLength++
	case 2:
		code.Codes[code.CodeLength] = cg.OP_astore_2
		code.CodeLength++
	case 3:
		code.Codes[code.CodeLength] = cg.OP_astore_3
		code.CodeLength++
	default:
		if context.function.AutoVarForMultiReturn.Offset > 255 {
			panic("local var offset over 255")
		}
		code.Codes[code.CodeLength] = cg.OP_astore
		code.Codes[code.CodeLength+1] = byte(context.function.AutoVarForMultiReturn.Offset)
		code.CodeLength += 2
	}
}

func (m *MakeExpression) unPackArraylist(class *cg.ClassHighLevel, code *cg.AttributeCode, k int, typ *ast.VariableType, context *Context) (maxstack uint16) {
	m.buildLoadArrayListAutoVar(code, context) // local array list on stack
	switch k {
	case 0:
		code.Codes[code.CodeLength] = cg.OP_iconst_0
		code.CodeLength++
	case 1:
		code.Codes[code.CodeLength] = cg.OP_iconst_1
		code.CodeLength++
	case 2:
		code.Codes[code.CodeLength] = cg.OP_iconst_2
		code.CodeLength++
	case 3:
		code.Codes[code.CodeLength] = cg.OP_iconst_3
		code.CodeLength++
	case 4:
		code.Codes[code.CodeLength] = cg.OP_iconst_4
		code.CodeLength++
	case 5:
		code.Codes[code.CodeLength] = cg.OP_iconst_5
		code.CodeLength++
	default:
		if k > 127 {
			panic("over 127")
		}
		code.Codes[code.CodeLength] = cg.OP_bipush
		code.Codes[code.CodeLength+1] = byte(k)
		code.CodeLength += 2
	}
	maxstack = 2
	code.Codes[code.CodeLength] = cg.OP_invokevirtual
	class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
		Class:      java_arrylist_class,
		Name:       "get",
		Descriptor: "(I)Ljava/lang/Object;",
	}, code.Codes[code.CodeLength+1:code.CodeLength+3])
	code.CodeLength += 3
	switch typ.Typ {
	case ast.VARIABLE_TYPE_BOOL:
		fallthrough
	case ast.VARIABLE_TYPE_BYTE:
		fallthrough
	case ast.VARIABLE_TYPE_SHORT:
		fallthrough
	case ast.VARIABLE_TYPE_INT:
		//cast to real object
		code.Codes[code.CodeLength] = cg.OP_checkcast
		class.InsertClassConst(java_integer_class, code.Codes[code.CodeLength+1:code.CodeLength+3])
		code.CodeLength += 3
		code.Codes[code.CodeLength] = cg.OP_invokevirtual
		class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
			Class:      java_integer_class,
			Name:       "intValue",
			Descriptor: "()I",
		}, code.Codes[code.CodeLength+1:code.CodeLength+3])
		code.CodeLength += 3
	case ast.VARIABLE_TYPE_LONG:
		//cast to real object
		code.Codes[code.CodeLength] = cg.OP_checkcast
		class.InsertClassConst(java_long_class, code.Codes[code.CodeLength+1:code.CodeLength+3])
		code.CodeLength += 3
		code.Codes[code.CodeLength] = cg.OP_invokevirtual
		class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
			Class:      java_long_class,
			Name:       "longValue",
			Descriptor: "()J",
		}, code.Codes[code.CodeLength+1:code.CodeLength+3])
		code.CodeLength += 3
	case ast.VARIABLE_TYPE_FLOAT:
		//cast to real object
		code.Codes[code.CodeLength] = cg.OP_checkcast
		class.InsertClassConst(java_float_class, code.Codes[code.CodeLength+1:code.CodeLength+3])
		code.CodeLength += 3
		code.Codes[code.CodeLength] = cg.OP_invokevirtual
		class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
			Class:      java_float_class,
			Name:       "floatValue",
			Descriptor: "()F",
		}, code.Codes[code.CodeLength+1:code.CodeLength+3])
		code.CodeLength += 3
	case ast.VARIABLE_TYPE_DOUBLE:
		//cast to real object
		code.Codes[code.CodeLength] = cg.OP_checkcast
		class.InsertClassConst("java/lang/Double", code.Codes[code.CodeLength+1:code.CodeLength+3])
		code.CodeLength += 3
		code.Codes[code.CodeLength] = cg.OP_invokevirtual
		class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
			Class:      java_double_class,
			Name:       "doubleValue",
			Descriptor: "()D",
		}, code.Codes[code.CodeLength+1:code.CodeLength+3])
		code.CodeLength += 3
	case ast.VARIABLE_TYPE_STRING:
	case ast.VARIABLE_TYPE_OBJECT:
	case ast.VARIABLE_TYPE_ARRAY_INSTANCE:
	}
	return
}

func (m *MakeExpression) controlStack2FitAssign(code *cg.AttributeCode, op []byte, stackTopType *ast.VariableType) (increment uint16) {
	// no object after value,just dup top
	if op[0] == cg.OP_istore || // 将栈顶 int 型数值存入指定局部变量。
		op[0] == cg.OP_lstore || //将栈顶 long 型数值存入指定局部变量。
		op[0] == cg.OP_fstore || //将栈顶 float 型数值存入指定局部变量。
		op[0] == cg.OP_dstore || //将栈顶 double 型数值存入指定局部变量。
		op[0] == cg.OP_astore || // 将栈顶引用型数值存入指定局部变量。
		op[0] == cg.OP_istore_0 || //将栈顶 int 型数值存入第一个局部变量。
		op[0] == cg.OP_istore_1 || // 将栈顶 int 型数值存入第二个局部变量。
		op[0] == cg.OP_istore_2 || //将栈顶 int 型数值存入第三个局部变量。
		op[0] == cg.OP_istore_3 || // 将栈顶 int 型数值存入第四个局部变量。
		op[0] == cg.OP_lstore_0 || //将栈顶 long 型数值存入第一个局部变量。
		op[0] == cg.OP_lstore_1 || // 将栈顶 long 型数值存入第二个局部变量。
		op[0] == cg.OP_lstore_2 || //将栈顶 long 型数值存入第三个局部变量。
		op[0] == cg.OP_lstore_3 || // 将栈顶 long 型数值存入第四个局部变量。
		op[0] == cg.OP_fstore_0 || //将栈顶 float 型数值存入第一个局部变量。
		op[0] == cg.OP_fstore_1 || //将栈顶 float 型数值存入第二个局部变量。
		op[0] == cg.OP_fstore_2 || //将栈顶 float 型数值存入第三个局部变量。
		op[0] == cg.OP_fstore_3 || //将栈顶 float 型数值存入第四个局部变量。
		op[0] == cg.OP_dstore_0 || //将栈顶 double 型数值存入第一个局部变量。
		op[0] == cg.OP_dstore_1 || //将栈顶 double 型数值存入第二个局部变量。
		op[0] == cg.OP_dstore_2 || // 将栈顶 double 型数值存入第三个局部变量。
		op[0] == cg.OP_dstore_3 || //将栈顶 double 型数值存入第四个局部变量。
		op[0] == cg.OP_astore_0 || //将栈顶引用型数值存入第一个局部变量。
		op[0] == cg.OP_astore_1 || ///将栈顶引用型数值存入第二个局部变量。
		op[0] == cg.OP_astore_2 || //将栈顶引用型数值存入第三个局部变量
		op[0] == cg.OP_astore_3 ||
		op[0] == cg.OP_putstatic { //为指定的类的静态域赋值。
		if stackTopType.JvmSlotSize() == 1 {
			increment = 1
			code.Codes[code.CodeLength] = cg.OP_dup
		} else {
			code.Codes[code.CodeLength] = cg.OP_dup2
			increment = 2
		}
		code.CodeLength++
		return
	}
	if op[0] == cg.OP_putfield {
		if stackTopType.JvmSlotSize() == 1 {
			increment = 1
			code.Codes[code.CodeLength] = cg.OP_dup_x1
		} else {
			increment = 2
			code.Codes[code.CodeLength] = cg.OP_dup_x2
		}
		code.CodeLength++
		return
	}
	panic(111111111)
	return
}

/*
	stack is ... objectref value
*/
func (m *MakeExpression) pack2Object(class *cg.ClassHighLevel, code *cg.AttributeCode, t *ast.VariableType) {
	switch t.Typ {
	case ast.VARIABLE_TYPE_BOOL:
		code.Codes[code.CodeLength] = cg.OP_invokespecial
		class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
			Class:      "java/lang/Boolean",
			Name:       specail_method_init,
			Descriptor: "(Z)V",
		}, code.Codes[code.CodeLength+1:code.CodeLength+3])
		code.CodeLength += 3
	case ast.VARIABLE_TYPE_BYTE:
		fallthrough
	case ast.VARIABLE_TYPE_SHORT:
		fallthrough
	case ast.VARIABLE_TYPE_INT:
		code.Codes[code.CodeLength] = cg.OP_invokespecial
		class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
			Class:      java_integer_class,
			Name:       specail_method_init,
			Descriptor: "(I)V",
		}, code.Codes[code.CodeLength+1:code.CodeLength+3])
		code.CodeLength += 3
	case ast.VARIABLE_TYPE_LONG:
		code.Codes[code.CodeLength] = cg.OP_invokespecial
		class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
			Class:      java_long_class,
			Name:       specail_method_init,
			Descriptor: "(J)V",
		}, code.Codes[code.CodeLength+1:code.CodeLength+3])
		code.CodeLength += 3
	case ast.VARIABLE_TYPE_FLOAT:
		code.Codes[code.CodeLength] = cg.OP_invokespecial
		class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
			Class:      java_float_class,
			Name:       specail_method_init,
			Descriptor: "(F)V",
		}, code.Codes[code.CodeLength+1:code.CodeLength+3])
		code.CodeLength += 3
	case ast.VARIABLE_TYPE_DOUBLE:
		code.Codes[code.CodeLength] = cg.OP_invokespecial
		class.InsertMethodRefConst(cg.CONSTANT_Methodref_info_high_level{
			Class:      java_double_class,
			Name:       specail_method_init,
			Descriptor: "(D)V",
		}, code.Codes[code.CodeLength+1:code.CodeLength+3])
		code.CodeLength += 3
	}
}
