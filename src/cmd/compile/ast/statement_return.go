package ast

import (
	"fmt"
)

type StatementReturn struct {
	ResultOffset uint16
	Defers       []*StatementDefer
	Expressions  []*Expression
}

func (s *StatementReturn) mkDefers(b *Block) {
	if b.IsFunctionBlock == false { // not top block
		s.mkDefers(b.Outer) // recursive
	}
	if b.Defers != nil {
		s.Defers = append(s.Defers, b.Defers...)
	}
}

func (s *StatementReturn) check(b *Block) []error {
	s.mkDefers(b)
	if len(s.Expressions) == 0 { // always ok
		return nil
	}
	errs := make([]error, 0)
	returnValueTypes := checkExpressions(b, s.Expressions, &errs, false)
	rs := b.InheritedAttribute.Function.Type.ReturnList
	pos := s.Expressions[len(s.Expressions)-1].Pos
	if len(returnValueTypes) < len(rs) {
		errs = append(errs, fmt.Errorf("%s too few arguments to return", errMsgPrefix(pos)))
	} else if len(returnValueTypes) > len(rs) {
		errs = append(errs, fmt.Errorf("%s too many arguments to return", errMsgPrefix(pos)))
	}
	convertExpressionsToNeeds(s.Expressions,
		b.InheritedAttribute.Function.Type.mkReturnTypes(s.Expressions[0].Pos), returnValueTypes)
	for k, v := range rs {
		if k < len(returnValueTypes) && returnValueTypes[k] != nil {
			if false == v.Type.Equal(&errs, returnValueTypes[k]) {
				errs = append(errs, fmt.Errorf("%s cannot use '%s' as '%s' to return",
					errMsgPrefix(returnValueTypes[k].Pos),
					returnValueTypes[k].TypeString(),
					v.Type.TypeString()))
			}
		}
	}
	return errs
}
