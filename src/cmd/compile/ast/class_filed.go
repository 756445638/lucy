package ast

import (
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/jvm/cg"
)

type ClassField struct {
	VariableDefinition
	LoadFromOutSide bool
}

func (f *ClassField) isStatic() bool {
	return (f.AccessFlags & cg.ACC_FIELD_STATIC) != 0
}

func (f *ClassField) isPublic() bool {
	return (f.AccessFlags & cg.ACC_FIELD_PUBLIC) != 0
}
func (f *ClassField) isProtected() bool {
	return (f.AccessFlags & cg.ACC_FIELD_PROTECTED) != 0
}
