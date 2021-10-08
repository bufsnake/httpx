package query

import "errors"

// 栈
type stack struct {
	stack []string
}

func NewStack() *stack {
	s := make([]string, 0)
	return &stack{stack: s}
}

func (s *stack) POP() (string, error) {
	if s.isEmpty() {
		return "", errors.New("stack is empty")
	}
	value := s.stack[len(s.stack)-1]
	newStack := make([]string, 0)
	for i := 0; i < len(s.stack)-1; i++ {
		newStack = append(newStack, s.stack[i])
	}
	s.stack = newStack
	return value, nil
}

func (s *stack) PUSH(value string) {
	newStack := make([]string, 0)
	newStack = append(newStack, s.stack...)
	newStack = append(newStack, value)
	s.stack = newStack
}

func (s *stack) isEmpty() bool {
	if len(s.stack) == 0 {
		return true
	}
	return false
}
