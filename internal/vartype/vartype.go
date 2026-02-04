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

// Variable represents a generic type wrapper that holds a value and tracks its initialization state.
type Variable[T any] struct {
	value T
	isset bool
}

// NewVariable creates and returns a new Variable instance initialized with the provided value.
func NewVariable[T any](value T) Variable[T] {
	return Variable[T]{
		isset: true,
		value: value,
	}
}

// Reset clears the value of the Variable and marks it as uninitialized.
func (v *Variable[T]) Reset() {
	var newVal T
	v.value = newVal
	v.isset = false
}

// Value retrieves the current value stored in the Variable.
func (v *Variable[T]) Value() T {
	return v.value
}

// Set assigns the provided value to the Variable and marks it as initialized.
func (v *Variable[T]) Set(val T) {
	v.value = val
	v.isset = true
}

// IsSet returns true if the Variable has been initialized with a value, otherwise false.
func (v *Variable[T]) IsSet() bool {
	return v.isset
}

// String returns a string representation of the Variable. If uninitialized, it returns a default placeholder message.
func (v Variable[T]) String() string {
	if !v.isset {
		return "Unsupported by weather provider"
	}
	return fmt.Sprint(v.value)
}
