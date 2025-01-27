package rego2sql

import (
	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/zclconf/go-cty/cty"
)

type Item struct {
	Node   *pg_query.Node
	Value  cty.Value
	Source string
}

type stack[T any] struct {
	queue []T
}

func newStack[T any]() *stack[T] {
	return &stack[T]{nil}
}

func (stack *stack[T]) Push(item T) {
	stack.queue = append([]T{item}, stack.queue...)
}

func (stack *stack[T]) Pop() T {
	if len(stack.queue) == 0 {
		panic("stack is empty")
	}

	item := stack.queue[len(stack.queue)-1]
	stack.queue = stack.queue[:len(stack.queue)-1]
	return item
}

func (stack *stack[T]) IsEmpty() bool {
	return len(stack.queue) == 0
}

func (stack *stack[T]) Len() int {
	return len(stack.queue)
}
