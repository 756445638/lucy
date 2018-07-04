package jvm

var (
	ArrayMetas                 = map[int]*ArrayMeta{}
	typeConverter              TypeConverterAndPrimitivePacker
	Descriptor                 Description
	LucyMethodSignatureParser  LucyMethodSignature
	LucyFieldSignatureParser   LucyFieldSignature
	LucyTypeAliasParser        LucyTypeAlias
	FunctionDefaultValueParser FunctionDefaultValueParse
	multiValuePacker           MultiValuePacker
	closure                    Closure
)

const (
	_ = iota
	LeftValueTypeLucyArray
	LeftValueTypeMap
	LeftValueTypeStoreLocalVar
	LeftValueTypePutStatic
	LeftValueTypePutField
	LeftValueTypeArray
)

const (
	specialMethodInit                 = "<init>"
	javaRootObjectArray               = "[Ljava/lang/Object;"
	javaStringClass                   = "java/lang/String"
	javaMethodHandleClass             = "java/lang/invoke/MethodHandle"
	javaRootClass                     = "java/lang/Object"
	javaMapClass                      = "java/util/HashMap"
	javaIntegerClass                  = "java/lang/Integer"
	javaFloatClass                    = "java/lang/Float"
	javaDoubleClass                   = "java/lang/Double"
	javaLongClass                     = "java/lang/Long"
	javaIndexOutOfRangeExceptionClass = "java/lang/ArrayIndexOutOfBoundsException"
	javaStringBuilderClass            = "java/lang/StringBuilder"
	throwableClass                    = "java/lang/Throwable"
	javaPrintStreamClass              = "java/io/PrintStream"
)
