// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package vartype

import (
	"fmt"
)

type VarFloat64 = Variable[float64]
type VarInt = Variable[int]
type VarBool = Variable[bool]

type Variable[T any] struct {
	value T
	isset bool
}

func NewVariable[T any](value T) Variable[T] {
	return Variable[T]{
		isset: true,
		value: value,
	}
}

func (v *Variable[T]) Reset() {
	var newVal T
	v.value = newVal
	v.isset = false
}

func (v *Variable[T]) Value() T {
	return v.value
}

func (v *Variable[T]) Set(val T) {
	v.value = val
	v.isset = true
}

func (v *Variable[T]) IsSet() bool {
	return v.isset
}

func (v Variable[T]) String() string {
	if !v.isset {
		return "Unsupported"
	}
	return fmt.Sprint(v.value)
}
