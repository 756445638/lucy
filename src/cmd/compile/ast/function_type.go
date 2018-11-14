package ast

import (
	"errors"
	"fmt"
)

type FunctionType struct {
	TemplateNames    []*NameWithPos
	TemplateNamesMap map[string]*Pos
	ParameterList    ParameterList
	ReturnList       ReturnList
	VArgs            *Variable
}

func (this *FunctionType) reDefineReturnVarOrParameter(v *Variable) error {
	if nil == this.searchName(v.Name) {
		return nil
	}
	return fmt.Errorf("%s redefine parameter or return var", v.Pos.ErrMsgPrefix())
}
func (this *FunctionType) CheckTemplateNameDuplication() []error {
	errs := []error{}
	m := make(map[string]*Pos)
	for _, v := range this.TemplateNames {
		if p, ok := m[v.Name]; ok {
			errMsg :=
				fmt.Sprintf("%s duplicated name '%s' , first declaraed at:\n",
					v.Pos.ErrMsgPrefix(), v.Name)
			errMsg += fmt.Sprintf("\t%s\n", p.ErrMsgPrefix())
			errs = append(errs, errors.New(errMsg))
			continue
		}
		m[v.Name] = v.Pos
	}
	this.TemplateNamesMap = m
	return errs
}

func (this *FunctionType) haveTemplateName(name string) bool {
	_, ok := this.TemplateNamesMap[name]
	return ok
}

type ParameterList []*Variable
type ReturnList []*Variable

func (this *FunctionType) Clone() (ret *FunctionType) {
	ret = &FunctionType{}
	ret.ParameterList = make(ParameterList, len(this.ParameterList))
	for k, _ := range ret.ParameterList {
		p := &Variable{}
		*p = *this.ParameterList[k]
		p.Type = this.ParameterList[k].Type.Clone()
		ret.ParameterList[k] = p
	}
	ret.ReturnList = make(ReturnList, len(this.ReturnList))
	for k, _ := range ret.ReturnList {
		p := &Variable{}
		*p = *this.ReturnList[k]
		p.Type = this.ReturnList[k].Type.Clone()
		ret.ReturnList[k] = p
	}
	return
}
func (this *FunctionType) TypeString() string {
	s := "("
	for k, v := range this.ParameterList {
		if v.Name != "" {
			s += v.Name + " "
		}
		s += v.Type.TypeString()
		if v.DefaultValueExpression != nil {
			s += " = " + v.DefaultValueExpression.Op
		}
		if k != len(this.ParameterList)-1 {
			s += ","
		}
	}
	if this.VArgs != nil {
		if len(this.ParameterList) > 0 {
			s += ","
		}
		if this.VArgs.Name != "" {
			s += this.VArgs.Name + " "
		}
		s += this.VArgs.Type.TypeString()
	}
	s += ")"
	if this.VoidReturn() == false {
		s += "->( "
		for k, v := range this.ReturnList {
			if v.Name != "" {
				s += v.Name + " "
			}
			s += v.Type.TypeString()
			if k != len(this.ReturnList)-1 {
				s += ","
			}
		}
		s += ")"
	}
	return s
}

func (this *FunctionType) searchName(name string) *Variable {
	if name == "" {
		return nil
	}
	for _, v := range this.ParameterList {
		if name == v.Name {
			return v
		}
	}
	if this.VoidReturn() == false {
		for _, v := range this.ReturnList {
			if name == v.Name {
				return v
			}
		}
	}
	return nil
}

func (this *FunctionType) equal(compare *FunctionType) bool {

	if len(this.ParameterList) != len(compare.ParameterList) {
		return false
	}

	for k, v := range this.ParameterList {
		if false == v.Type.Equal(compare.ParameterList[k].Type) {
			return false
		}
	}
	if (this.VArgs == nil) != (compare.VArgs == nil) {
		return false
	}

	if this.VArgs != nil {
		if this.VArgs.Type.Equal(compare.VArgs.Type) == false {
			return false
		}
	}
	if this.VoidReturn() != compare.VoidReturn() {
		return false
	}

	if this.VoidReturn() == false {
		for k, v := range this.ReturnList {
			if false == v.Type.Equal(compare.ReturnList[k].Type) {
				return false
			}
		}
	}
	return true
}

func (this *FunctionType) callHave(ts []*Type) string {
	s := "("
	for k, v := range ts {
		if v == nil {
			continue
		}
		if v.Name != "" {
			s += v.Name + " "
		}
		s += v.TypeString()
		if k != len(ts)-1 {
			s += ","
		}
	}
	s += ")"
	return s
}

func (this *FunctionType) VoidReturn() bool {
	return len(this.ReturnList) == 0 ||
		this.ReturnList[0].Type.Type == VariableTypeVoid
}

func (this FunctionType) mkCallReturnTypes(pos *Pos) []*Type {
	if this.ReturnList == nil || len(this.ReturnList) == 0 {
		t := &Type{}
		t.Type = VariableTypeVoid // means no return ;
		t.Pos = pos
		return []*Type{t}
	}
	ret := make([]*Type, len(this.ReturnList))
	for k, v := range this.ReturnList {
		ret[k] = v.Type.Clone()
		ret[k].Pos = pos
	}
	return ret
}

func (this FunctionType) getParameterTypes() []*Type {
	ret := make([]*Type, len(this.ParameterList))
	for k, v := range this.ParameterList {
		ret[k] = v.Type
	}
	return ret
}

func (this *FunctionType) callArgsHasNoNil(ts []*Type) bool {
	for _, t := range ts {
		if t == nil {
			return false
		}
	}
	return true
}

func (this *FunctionType) fitArgs(
	from *Pos,
	args *CallArgs,
	callArgsTypes []*Type,
	f *Function) (vArgs *CallVariableArgs, err error) {
	if this.VArgs != nil {
		vArgs = &CallVariableArgs{}
		vArgs.NoArgs = true
		vArgs.Type = this.VArgs.Type
	}
	var haveAndWant string
	if this.callArgsHasNoNil(callArgsTypes) {
		haveAndWant = fmt.Sprintf("\thave %s\n", this.callHave(callArgsTypes))
		haveAndWant += fmt.Sprintf("\twant %s\n", this.wantArgs())
	}
	errs := []error{}
	if len(callArgsTypes) > len(this.ParameterList) {
		if this.VArgs == nil {
			errMsg := fmt.Sprintf("%s too many paramaters to call\n", errMsgPrefix(from))
			errMsg += haveAndWant
			err = fmt.Errorf(errMsg)
			return
		}
		v := this.VArgs
		for _, t := range callArgsTypes[len(this.ParameterList):] {
			if t == nil { // some error before
				return
			}
			if t.IsVariableArgs {
				if len(callArgsTypes[len(this.ParameterList):]) > 1 {
					errMsg := fmt.Sprintf("%s too many argument to call\n",
						errMsgPrefix(t.Pos))
					errMsg += haveAndWant
					err = fmt.Errorf(errMsg)
					return
				}
				if false == v.Type.assignAble(&errs, t) {
					err = fmt.Errorf("%s cannot use '%s' as '%s'",
						errMsgPrefix(t.Pos),
						t.TypeString(), v.Type.TypeString())
					return
				}
				vArgs.PackArray2VArgs = true
				continue
			}
			if false == v.Type.Array.assignAble(&errs, t) {
				err = fmt.Errorf("%s cannot use '%s' as '%s'",
					errMsgPrefix(t.Pos),
					t.TypeString(), v.Type.TypeString())
				return
			}
		}
		vArgs.NoArgs = false
		k := len(this.ParameterList)
		vArgs.Length = len(callArgsTypes) - k
		vArgs.Expressions = (*args)[k:]
		*args = (*args)[:k]
		vArgs.Length = len(callArgsTypes) - k
	}
	if len(callArgsTypes) < len(this.ParameterList) {
		if f != nil && f.HaveDefaultValue && len(callArgsTypes) >= f.DefaultValueStartAt {
			for i := len(callArgsTypes); i < len(f.Type.ParameterList); i++ {
				*args = append(*args, f.Type.ParameterList[i].DefaultValueExpression)
			}
		} else { // no default value
			errMsg := fmt.Sprintf("%s too few paramaters to call\n", errMsgPrefix(from))
			errMsg += haveAndWant
			err = fmt.Errorf(errMsg)
			return
		}
	}

	for k, v := range this.ParameterList {
		if k < len(callArgsTypes) && callArgsTypes[k] != nil {
			if v.Type.assignAble(&errs, callArgsTypes[k]) {
				continue
			}
			//TODO :: convert or not ???
			errMsg := fmt.Sprintf("%s cannot use '%s' as '%s'",
				errMsgPrefix(callArgsTypes[k].Pos),
				callArgsTypes[k].TypeString(), v.Type.TypeString())
			err = fmt.Errorf(errMsg)
			return
		}
	}

	return
}

type CallVariableArgs struct {
	Expressions []*Expression
	Length      int
	/*
		a := new int[](10)
		print(a...)
	*/
	PackArray2VArgs bool
	NoArgs          bool
	Type            *Type
}

func (this *FunctionType) wantArgs() string {
	s := "("
	for k, v := range this.ParameterList {
		s += v.Name + " "
		s += v.Type.TypeString()
		if k != len(this.ParameterList)-1 {
			s += ","
		}
	}
	if this.VArgs != nil {
		if len(this.ParameterList) > 0 {
			s += ","
		}
		s += this.VArgs.Name + " "
		s += this.VArgs.Type.TypeString()
	}
	s += ")"
	return s
}
