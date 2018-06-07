package jvm

import (
	"encoding/binary"

	"gitee.com/yuyang-fine/lucy/src/cmd/compile/ast"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/jvm/cg"
)

func (m *MakeClass) buildIfStatement(class *cg.ClassHighLevel, code *cg.AttributeCode, s *ast.StatementIF, context *Context, state *StackMapState) (maxstack uint16) {
	var es []*cg.JumpBackPatch
	conditionState := (&StackMapState{}).FromLast(state)
	defer state.addTop(conditionState)
	for _, v := range s.PreExpressions {
		stack, _ := m.MakeExpression.build(class, code, v, context, conditionState)
		if stack > maxstack {
			maxstack = stack
		}
	}
	var IfState *StackMapState
	if s.Block.HaveVariableDefinition() {
		IfState = (&StackMapState{}).FromLast(conditionState)
	} else {
		IfState = conditionState
	}
	maxstack, es = m.MakeExpression.build(class, code, s.Condition, context, IfState)
	if len(es) > 0 {
		backPatchEs(es, code.CodeLength)
		IfState.pushStack(class, s.Condition.Value)
		context.MakeStackMap(code, IfState, code.CodeLength)
		IfState.popStack(1) // must be bool expression
	}
	code.Codes[code.CodeLength] = cg.OP_ifeq
	codelength := code.CodeLength
	exit := code.Codes[code.CodeLength+1 : code.CodeLength+3]
	code.CodeLength += 3
	m.buildBlock(class, code, &s.Block, context, IfState)
	conditionState.addTop(IfState)
	if s.Block.DeadEnding == false {
		s.BackPatchs = append(s.BackPatchs, (&cg.JumpBackPatch{}).FromCode(cg.OP_goto, code))
	}
	for _, v := range s.ElseIfList {
		context.MakeStackMap(code, conditionState, code.CodeLength) // state is not change,all block var should be access from outside
		binary.BigEndian.PutUint16(exit, uint16(code.CodeLength-codelength))
		var elseIfState *StackMapState
		if v.Block.HaveVariableDefinition() {
			elseIfState = (&StackMapState{}).FromLast(conditionState)
		} else {
			elseIfState = conditionState
		}
		stack, es := m.MakeExpression.build(class, code, v.Condition, context, elseIfState)
		if len(es) > 0 {
			backPatchEs(es, code.CodeLength)
			elseIfState.pushStack(class, s.Condition.Value)
			context.MakeStackMap(code, elseIfState, code.CodeLength)
			elseIfState.popStack(1)
		}
		if stack > maxstack {
			maxstack = stack
		}
		code.Codes[code.CodeLength] = cg.OP_ifeq
		codelength = code.CodeLength
		exit = code.Codes[code.CodeLength+1 : code.CodeLength+3]
		code.CodeLength += 3
		m.buildBlock(class, code, v.Block, context, elseIfState)

		if v.Block.DeadEnding == false {
			s.BackPatchs = append(s.BackPatchs, (&cg.JumpBackPatch{}).FromCode(cg.OP_goto, code))
		}

		// when done
		conditionState.addTop(elseIfState)
	}
	context.MakeStackMap(code, conditionState, code.CodeLength)
	binary.BigEndian.PutUint16(exit, uint16(code.CodeLength-codelength))
	if s.ElseBlock != nil {
		var elseState *StackMapState
		if s.ElseBlock.HaveVariableDefinition() {
			elseState = (&StackMapState{}).FromLast(conditionState)
		} else {
			elseState = conditionState
		}
		m.buildBlock(class, code, s.ElseBlock, context, elseState)
		conditionState.addTop(elseState)
		if s.ElseBlock.DeadEnding == false {
			s.BackPatchs = append(s.BackPatchs, (&cg.JumpBackPatch{}).FromCode(cg.OP_goto, code))
		}
	}

	return
}
