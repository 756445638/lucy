package ast

import (
	"fmt"

	"github.com/756445638/lucy/src/cmd/compile/jvm/cg"
	"github.com/756445638/lucy/src/cmd/compile/jvm/class_json"
)

type Function struct {
	isPackageBlockFunction     bool
	callchecker                CallChecker // used in build function
	ClassMethod                *cg.MethodHighLevel
	IsGlobal                   bool
	Isbuildin                  bool
	Used                       bool
	AccessFlags                uint16 // public private or protected
	Typ                        *FunctionType
	ClosureVars                ClosureVars
	Name                       string // if name is nil string,means no name function
	Block                      *Block
	Pos                        *Pos
	Descriptor                 string
	Signature                  *class_json.MethodSignature
	VariableType               VariableType
	Varoffset                  uint16
	ArrayListVarForMultiReturn *ArrayListVarForMultiReturn
}

func (f *Function) mkArrayListVarForMultiReturn() {
	if f.ArrayListVarForMultiReturn == nil {
		t := &ArrayListVarForMultiReturn{}
		t.Offset = f.Varoffset
		f.ArrayListVarForMultiReturn = t
		f.Varoffset++
	}
}

type ArrayListVarForMultiReturn struct {
	Offset uint16
}

func (f *Function) IsClosureFunction() bool {
	return f.ClosureVars.NotEmpty()
}

func (f *Function) readableMsg() string {
	s := "fn" + f.Name + "("
	for k, v := range f.Typ.ParameterList {
		s += v.Name + " " + v.Typ.TypeString()
		if k != len(f.Typ.ParameterList)-1 {
			s += ","
		}
	}
	s += ")"
	if len(f.Typ.ReturnList) > 0 {
		s += "->"
		s += "("
		for k, v := range f.Typ.ReturnList {
			s += v.Name + " " + v.Typ.TypeString() + ","
			if k != len(f.Typ.ReturnList)-1 {
				s += ","
			}
		}
		s += ")"
	}
	return s
}

func (f *Function) MkVariableType() {
	f.VariableType.Typ = VARIABLE_TYPE_FUNCTION
	f.VariableType.Function = f
}

func (f *Function) MkDescriptor() string {
	if f.Name == MAIN_FUNCTION_NAME {
		return "([Ljava/lang/String;)V"
	}
	s := "("
	for _, v := range f.Typ.ParameterList {
		s += v.NameWithType.Typ.Descriptor()
	}
	s += ")"
	if len(f.Typ.ReturnList) == 0 {
		s += "V"
	} else if len(f.Typ.ReturnList) == 1 {
		s += f.Typ.ReturnList[0].Typ.Descriptor()
	} else {
		s += "Ljava/util/ArrayList;" //always this type
	}
	return s
}

func (f *Function) checkBlock(errs *[]error) {
	f.mkLastRetrunStatement()
	*errs = append(*errs, f.Block.check(nil)...)
}

func (f *Function) check(b *Block) []error {
	errs := make([]error, 0)
	f.Block.inherite(b)
	f.Block.InheritedAttribute.function = f
	f.checkParaMeterAndRetuns(&errs)
	f.checkBlock(&errs)
	return errs
}

func (f *Function) mkLastRetrunStatement() {
	if len(f.Block.Statements) == 0 ||
		(f.Block.Statements[len(f.Block.Statements)-1].Typ != STATEMENT_TYPE_RETURN) {
		s := &StatementReturn{}
		es := []*Expression{}
		for _, v := range f.Typ.ReturnList {
			identifer := &ExpressionIdentifer{}
			identifer.Name = v.Name
			es = append(es, &Expression{
				Typ:  EXPRESSION_TYPE_IDENTIFIER,
				Data: identifer,
			})
		}
		f.Block.Statements = append(f.Block.Statements, &Statement{Typ: STATEMENT_TYPE_RETURN, StatementReturn: s})
	}
}

func (f *Function) checkParaMeterAndRetuns(errs *[]error) {
	if f.Name == MAIN_FUNCTION_NAME {
		if len(f.Typ.ParameterList) > 0 {
			*errs = append(*errs, fmt.Errorf("%s function main must not taken no parameters ", errMsgPrefix(f.Pos)))
		}
		if len(f.Typ.ReturnList) > 0 {
			*errs = append(*errs, fmt.Errorf("%s function main must have no return values ", errMsgPrefix(f.Pos)))
		}
		f.Varoffset++
		return
	}

	var err error
	for _, v := range f.Typ.ParameterList {
		v.IsFunctionParameter = true
		err = v.Typ.resolve(f.Block)
		if err != nil {
			*errs = append(*errs, fmt.Errorf("%s %s", errMsgPrefix(v.Pos), err.Error()))
		}
		err = f.Block.insert(v.Name, v.Pos, v)
		if err != nil {
			*errs = append(*errs, err)
			continue
		}
	}
	//handler return
	for _, v := range f.Typ.ReturnList {
		err = v.Typ.resolve(f.Block)
		if err != nil {
			*errs = append(*errs, err)
		}
		err = f.Block.insert(v.Name, v.Pos, v)
		if err != nil {
			*errs = append(*errs, fmt.Errorf("%s err:%v", errMsgPrefix(v.Pos), err))
		}
	}
}

type FunctionType struct {
	ParameterList ParameterList
	ReturnList    ReturnList
}

type ParameterList []*VariableDefinition
type ReturnList []*VariableDefinition

func (r ReturnList) retTypes(pos *Pos) []*VariableType {
	if r == nil || len(r) == 0 {
		t := &VariableType{}
		t.Typ = VARIABLE_TYPE_VOID // means no return;
		return []*VariableType{t}
	}
	ret := make([]*VariableType, len(r))
	for k, v := range r {
		ret[k] = v.Typ.Clone()
		ret[k].Pos = pos
	}
	return ret
}
