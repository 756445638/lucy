package cg

import (
	"encoding/binary"
)

type AttributeTemplateFunction struct {
	Name        string
	Filename    string
	StartLine   uint16
	StartColumn uint16
	Code        string
	Descriptor  string
}

func (a *AttributeTemplateFunction) FromBytes(class *Class, bs []byte) {
	if len(bs) != 12 {
		panic("not 12")
	}
	a.Name = string(class.ConstPool[binary.BigEndian.Uint16(bs)].Info)
	a.Filename = string(class.ConstPool[binary.BigEndian.Uint16(bs[2:])].Info)
	a.StartLine = binary.BigEndian.Uint16(bs[4:])
	a.StartColumn = binary.BigEndian.Uint16(bs[6:])
	a.Code = string(class.ConstPool[binary.BigEndian.Uint16(bs[8:])].Info)
	a.Descriptor = string(class.ConstPool[binary.BigEndian.Uint16(bs[10:])].Info)
}

func (a *AttributeTemplateFunction) ToAttributeInfo(class *Class) *AttributeInfo {
	info := &AttributeInfo{}
	info.NameIndex = class.insertUtf8Const(ATTRIBUTE_NAME_LUCY_TEMPLATE_FUNCTION)
	info.attributeLength = 12
	info.Info = make([]byte, info.attributeLength)
	binary.BigEndian.PutUint16(info.Info, class.insertUtf8Const(a.Name))
	binary.BigEndian.PutUint16(info.Info[2:], class.insertUtf8Const(a.Filename))
	binary.BigEndian.PutUint16(info.Info[4:], a.StartLine)
	binary.BigEndian.PutUint16(info.Info[6:], a.StartColumn)
	binary.BigEndian.PutUint16(info.Info[8:], class.insertUtf8Const(a.Code))
	binary.BigEndian.PutUint16(info.Info[10:], class.insertUtf8Const(a.Descriptor))
	return info
}
