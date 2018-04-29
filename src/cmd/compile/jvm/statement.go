package jvm

import (
	//"fmt"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/ast"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/jvm/cg"
)

func (m *MakeClass) buildStatement(class *cg.ClassHighLevel, code *cg.AttributeCode, b *ast.Block, s *ast.Statement,
	context *Context, state *StackMapState) (maxstack uint16) {
	//fmt.Println(s.StatementName())
	//	fmt.Printf("compile:%s %v\n", s.Pos.Filename, s.Pos.StartLine)
	switch s.Typ {
	case ast.STATEMENT_TYPE_EXPRESSION:
		if s.Expression.Typ == ast.EXPRESSION_TYPE_FUNCTION {
			return m.buildFunctionExpression(class, code, s.Expression, context, state)
		}
		maxstack, _ = m.MakeExpression.build(class, code, s.Expression, context, state)
	case ast.STATEMENT_TYPE_IF:
		maxstack = m.buildIfStatement(class, code, s.StatementIf, context, state)
		if len(s.StatementIf.BackPatchs) > 0 {
			backPatchEs(s.StatementIf.BackPatchs, code.CodeLength)
			context.MakeStackMap(code, state, code.CodeLength)
		}
	case ast.STATEMENT_TYPE_BLOCK: //new
		ss := (&StackMapState{}).FromLast(state)
		m.buildBlock(class, code, s.Block, context, ss)
		state.addTop(ss)
	case ast.STATEMENT_TYPE_FOR:
		maxstack = m.buildForStatement(class, code, s.StatementFor, context, state)
		if len(s.StatementFor.BackPatchs) > 0 {
			backPatchEs(s.StatementFor.BackPatchs, code.CodeLength)
			context.MakeStackMap(code, state, code.CodeLength)
		}
		if len(s.StatementFor.ContinueBackPatchs) > 0 {
			// stack map is already maked
			backPatchEs(s.StatementFor.ContinueBackPatchs, s.StatementFor.ContinueOPOffset)
		}
	case ast.STATEMENT_TYPE_CONTINUE:
		if b.Defers != nil && len(b.Defers) > 0 {
			index := len(b.Defers) - 1
			for index >= 0 {
				ss := (&StackMapState{}).FromLast(state)
				m.buildBlock(class, code, &b.Defers[index].Block, context, state)
				index--
				state.addTop(ss)
			}
		}
		s.StatementContinue.StatementFor.ContinueBackPatchs = append(s.StatementContinue.StatementFor.ContinueBackPatchs,
			(&cg.JumpBackPatch{}).FromCode(cg.OP_goto, code))
	case ast.STATEMENT_TYPE_BREAK:
		if b.Defers != nil && len(b.Defers) > 0 {
			index := len(b.Defers) - 1
			for index >= 0 {
				ss := (&StackMapState{}).FromLast(state)
				m.buildBlock(class, code, &b.Defers[index].Block, context, state)
				index--
				state.addTop(ss)
			}
		}
		code.Codes[code.CodeLength] = cg.OP_goto
		b := (&cg.JumpBackPatch{}).FromCode(cg.OP_goto, code)
		if s.StatementBreak.StatementFor != nil {
			s.StatementBreak.StatementFor.BackPatchs = append(s.StatementBreak.StatementFor.BackPatchs, b)
		} else { // switch
			s.StatementBreak.StatementSwitch.BackPatchs = append(s.StatementBreak.StatementSwitch.BackPatchs, b)
		}
	case ast.STATEMENT_TYPE_RETURN:
		maxstack = m.buildReturnStatement(class, code, s.StatementReturn, context, state)
	case ast.STATEMENT_TYPE_SWITCH:
		maxstack = m.buildSwitchStatement(class, code, s.StatementSwitch, context, state)
		backPatchEs(s.StatementSwitch.BackPatchs, code.CodeLength)
	case ast.STATEMENT_TYPE_SKIP: // skip this block
		code.Codes[code.CodeLength] = cg.OP_return
		code.CodeLength++
	case ast.STATEMENT_TYPE_GOTO:
		b := (&cg.JumpBackPatch{}).FromCode(cg.OP_goto, code)
		s.StatementGoto.StatementLable.BackPatches = append(s.StatementGoto.StatementLable.BackPatches, b)
	case ast.STATEMENT_TYPE_LABLE:
		if len(s.StatmentLable.BackPatches) > 0 {
			backPatchEs(s.StatmentLable.BackPatches, code.CodeLength) // back patch
			context.MakeStackMap(code, state, code.CodeLength)
		}
	case ast.STATEMENT_TYPE_DEFER: // nothing to do  ,defer will do after block is compiled
		s.Defer.StartPc = code.CodeLength
		s.Defer.StackMapState = (&StackMapState{}).FromLast(state)
	}
	return
}
