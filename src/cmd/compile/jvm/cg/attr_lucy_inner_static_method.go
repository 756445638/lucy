package cg

type AttributeLucyInnerStaticMethod struct {
}

func (a *AttributeLucyInnerStaticMethod) ToAttributeInfo(class *Class) *AttributeInfo {
	ret := &AttributeInfo{}
	ret.NameIndex = class.insertUtfConst(ATTRIBUTE_NAME_LUCY_INNER_STATIC_METHOD)
	return ret
}