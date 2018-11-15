package jvm

import "gitee.com/yuyang-fine/lucy/src/cmd/compile/ast"

var (
	ArrayMetas                = map[ast.VariableTypeKind]*ArrayMeta{}
	typeConverter             TypeConverterAndPrimitivePacker
	Descriptor                Description
	LucyMethodSignatureParser LucyMethodSignature
	LucyFieldSignatureParser  LucyFieldSignature
	LucyTypeAliasParser       LucyTypeAlias
	DefaultValueParser        DefaultValueParse
	closure                   Closure
)

type LeftValueKind int

const (
	_ LeftValueKind = iota
	LeftValueKindLucyArray
	LeftValueKindMap
	LeftValueKindLocalVar
	LeftValueKindPutStatic
	LeftValueKindPutField
	LeftValueKindArray
)

const (
	methodHandleInvokeMethodName = "invoke"
	specialMethodInit            = "<init>"
	javaRootObjectArray          = "[Ljava/lang/Object;"
	javaStringClass              = "java/lang/String"
	javaExceptionClass           = "java/lang/Exception"
	javaMethodHandleClass        = "java/lang/invoke/MethodHandle"
	javaRootClass                = "java/lang/Object"
	javaIntegerClass             = "java/lang/Integer"
	javaFloatClass               = "java/lang/Float"
	javaDoubleClass              = "java/lang/Double"
	javaLongClass                = "java/lang/Long"
	throwableClass               = "java/lang/Throwable"
	javaPrintStreamClass         = "java/io/PrintStream"
	mapClass                     = "java/util/HashMap"
)
