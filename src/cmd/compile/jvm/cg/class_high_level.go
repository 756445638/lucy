package cg

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

type ClassHighLevel struct {
	Class                  Class
	SourceFiles            map[string]struct{} // one class file can be compile form multi
	Name                   string
	IsClosureFunctionClass bool
	MainClass              *ClassHighLevel
	InnerClasss            []*ClassHighLevel
	AccessFlags            uint16
	//FieldRefs              map[CONSTANT_Fieldref_info_high_level][][]byte
	MethodRefs map[CONSTANT_Methodref_info_high_level][][]byte
	//NameAndTypes map[CONSTANT_NameAndType_info_high_level][][]byte
	SuperClass string
	Interfaces []string
	Fields     map[string]*FiledHighLevel
	Methods    map[string][]*MethodHighLevel
}

/*
	new a method name,mksure it does exists before
*/
func (c *ClassHighLevel) NewFunctionName(prefix string) string {
	for i := 0; i < math.MaxInt16; i++ {
		name := prefix + fmt.Sprintf("%d", i)
		if _, ok := c.Methods[name]; ok == false {
			return name
		}
	}
	panic(1)
}
func (c *ClassHighLevel) InsertStringConst(s string, location []byte) {
	binary.BigEndian.PutUint16(location, c.Class.InsertStringConst(s))
}

func (c *ClassHighLevel) AppendMethod(ms ...*MethodHighLevel) {
	if c.Methods == nil {
		c.Methods = make(map[string][]*MethodHighLevel)
	}
	for _, v := range ms {
		if v.Name == "" {
			panic(1)
		}
		if _, ok := c.Methods[v.Name]; ok {
			c.Methods[v.Name] = append(c.Methods[v.Name], v)
		} else {
			c.Methods[v.Name] = []*MethodHighLevel{v}
		}
	}
}

type CONSTANT_NameAndType_info_high_level struct {
	Name string
	Type string
}

//func (c *ClassHighLevel) InsertNameAndTypeConst(nameAndType CONSTANT_NameAndType_info_high_level, location []byte) {
//	if c.NameAndTypes == nil {
//		c.NameAndTypes = make(map[CONSTANT_NameAndType_info_high_level][][]byte)
//	}
//	if _, ok := c.NameAndTypes[nameAndType]; ok {
//		c.NameAndTypes[nameAndType] = append(c.NameAndTypes[nameAndType], location)
//	} else {
//		c.NameAndTypes[nameAndType] = [][]byte{location}
//	}
//}

type CONSTANT_Methodref_info_high_level struct {
	Class      string
	Name       string
	Descriptor string
}

func (c *ClassHighLevel) InsertMethodRefConst(mr CONSTANT_Methodref_info_high_level, location []byte) {
	if c.MethodRefs == nil {
		c.MethodRefs = make(map[CONSTANT_Methodref_info_high_level][][]byte)
	}
	if _, ok := c.MethodRefs[mr]; ok {
		c.MethodRefs[mr] = append(c.MethodRefs[mr], location)
	} else {
		c.MethodRefs[mr] = [][]byte{location}
	}
}

type CONSTANT_Fieldref_info_high_level struct {
	Class      string
	Name       string
	Descriptor string
}

func (c *ClassHighLevel) InsertFieldRefConst(fr CONSTANT_Fieldref_info_high_level, location []byte) {
	binary.BigEndian.PutUint16(location, c.Class.InsertFieldRefConst(fr))
}
func (c *ClassHighLevel) InsertClassConst(classname string, location []byte) {

	binary.BigEndian.PutUint16(location, c.Class.InsertClassConst(classname))
}
func (c *ClassHighLevel) InsertIntConst(i int32, location []byte) {
	binary.BigEndian.PutUint16(location, c.Class.InsertIntConst(i))
}
func (c *ClassHighLevel) InsertLongConst(i int64, location []byte) {
	binary.BigEndian.PutUint16(location, c.Class.InsertLongConst(i))
}

func (c *ClassHighLevel) InsertFloatConst(f float32, location []byte) {
	binary.BigEndian.PutUint16(location, c.Class.InsertFloatConst(f))
}

func (c *ClassHighLevel) InsertDoubleConst(d float64, location []byte) {
	binary.BigEndian.PutUint16(location, c.Class.InsertDoubleConst(d))
}
func (c *ClassHighLevel) getSourceFile() string {
	s := ""
	for f, _ := range c.SourceFiles {
		s += f + " "
	}
	if s != "" {
		s = strings.TrimRight(s, " ")
	}
	return s
}
