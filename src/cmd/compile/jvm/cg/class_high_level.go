package cg

import (
	"encoding/binary"
	"fmt"
	"math"
)

type ClassHighLevel struct {
	Class             Class
	SourceFiles       map[string]struct{} // one class file can be compile form multi
	Name              string
	AccessFlags       uint16
	SuperClass        string
	Interfaces        []string
	Fields            map[string]*FieldHighLevel
	Methods           map[string][]*MethodHighLevel
	TriggerCLinit     *MethodHighLevel
	TemplateFunctions []*AttributeTemplateFunction
}

func (c *ClassHighLevel) InsertMethodRefConst(mr CONSTANT_Methodref_info_high_level, location []byte) {
	binary.BigEndian.PutUint16(location, c.Class.InsertMethodrefConst(mr))
}

/*
	new a method name,make sure it does exists before
*/
func (c *ClassHighLevel) NewFunctionName(prefix string) string {
	if c.Methods == nil || c.Methods[prefix] == nil {
		return prefix
	}
	for i := 0; i < math.MaxInt16; i++ {
		name := prefix + fmt.Sprintf("%d", i)
		if _, ok := c.Methods[name]; ok == false {
			return name
		}
	}
	panic("names over flow") // this is not happening
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
			panic("null name")
		}
		if _, ok := c.Methods[v.Name]; ok {
			c.Methods[v.Name] = append(c.Methods[v.Name], v)
		} else {
			c.Methods[v.Name] = []*MethodHighLevel{v}
		}
	}
}

func (c *ClassHighLevel) InsertInterfaceMethodrefConst(fr CONSTANT_InterfaceMethodref_info_high_level, location []byte) {
	binary.BigEndian.PutUint16(location, c.Class.InsertInterfaceMethodrefConst(fr))
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

/*
	source files
*/
func (c *ClassHighLevel) getSourceFile() string {
	s := ""
	last := len(c.SourceFiles) - 1
	i := 0
	for f, _ := range c.SourceFiles {
		s += f
		if i != last {
			s += "\n"
		}
		i++
	}
	return s
}
