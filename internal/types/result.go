// Package types provides core types and functional programming primitives for Antimoji.
package types

import "fmt"

// Result represents a computation that may fail, following functional programming principles.
// It encapsulates both success and error cases without using exceptions.
type Result[T any] struct {
	value T
	err   error
}

// Ok creates a successful Result containing the given value.
func Ok[T any](value T) Result[T] {
	return Result[T]{value: value}
}

// Err creates a failed Result containing the given error.
func Err[T any](err error) Result[T] {
	return Result[T]{err: err}
}

// IsOk returns true if the Result represents success.
func (r Result[T]) IsOk() bool {
	return r.err == nil
}

// IsErr returns true if the Result represents an error.
func (r Result[T]) IsErr() bool {
	return r.err != nil
}

// Unwrap returns the contained value if successful, or panics if the Result is an error.
// Use this only when you're certain the Result is Ok, prefer UnwrapOr or match pattern.
func (r Result[T]) Unwrap() T {
	if r.err != nil {
		panic(fmt.Sprintf("called Unwrap on an Err value: %v", r.err))
	}
	return r.value
}

// UnwrapOr returns the contained value if successful, or the default value if error.
func (r Result[T]) UnwrapOr(defaultValue T) T {
	if r.err != nil {
		return defaultValue
	}
	return r.value
}

// UnwrapOrElse returns the contained value if successful, or computes a default using the function.
func (r Result[T]) UnwrapOrElse(f func(error) T) T {
	if r.err != nil {
		return f(r.err)
	}
	return r.value
}

// Error returns the contained error, or nil if successful.
func (r Result[T]) Error() error {
	return r.err
}

// Value returns the contained value and error for standard Go error handling.
func (r Result[T]) Value() (T, error) {
	return r.value, r.err
}

// MapErr transforms the contained error if present, leaving successful values unchanged.
func (r Result[T]) MapErr(f func(error) error) Result[T] {
	if r.err == nil {
		return r
	}
	return Err[T](f(r.err))
}

// Or returns this Result if successful, otherwise returns the other Result.
func (r Result[T]) Or(other Result[T]) Result[T] {
	if r.err == nil {
		return r
	}
	return other
}

// OrElse returns this Result if successful, otherwise computes another Result.
func (r Result[T]) OrElse(f func(error) Result[T]) Result[T] {
	if r.err == nil {
		return r
	}
	return f(r.err)
}

// String implements fmt.Stringer for debugging.
func (r Result[T]) String() string {
	if r.err != nil {
		return fmt.Sprintf("Err(%v)", r.err)
	}
	return fmt.Sprintf("Ok(%v)", r.value)
}

// Match provides pattern matching on Result values.
func (r Result[T]) Match(onOk func(T), onErr func(error)) {
	if r.err != nil {
		onErr(r.err)
	} else {
		onOk(r.value)
	}
}

// TryFrom converts a standard Go (value, error) pair to a Result.
func TryFrom[T any](value T, err error) Result[T] {
	if err != nil {
		return Err[T](err)
	}
	return Ok(value)
}

// Map transforms the contained value if successful, leaving errors unchanged.
// This is a standalone function to work around Go's generic method limitations.
func Map[T, U any](r Result[T], f func(T) U) Result[U] {
	if r.err != nil {
		return Err[U](r.err)
	}
	return Ok(f(r.value))
}

// FlatMap applies a function that returns a Result, avoiding nested Results.
// Also known as "bind" or "chain" in functional programming.
func FlatMap[T, U any](r Result[T], f func(T) Result[U]) Result[U] {
	if r.err != nil {
		return Err[U](r.err)
	}
	return f(r.value)
}

// AndThen is an alias for FlatMap for better readability in chains.
func AndThen[T, U any](r Result[T], f func(T) Result[U]) Result[U] {
	return FlatMap(r, f)
}

// Collect converts a slice of Results into a Result of slice.
// If any Result is an error, returns the first error encountered.
func Collect[T any](results []Result[T]) Result[[]T] {
	values := make([]T, 0, len(results))
	for _, result := range results {
		if result.err != nil {
			return Err[[]T](result.err)
		}
		values = append(values, result.value)
	}
	return Ok(values)
}

// Partition separates a slice of Results into successful values and errors.
func Partition[T any](results []Result[T]) ([]T, []error) {
	var values []T
	var errors []error

	for _, result := range results {
		if result.err != nil {
			errors = append(errors, result.err)
		} else {
			values = append(values, result.value)
		}
	}

	return values, errors
}
