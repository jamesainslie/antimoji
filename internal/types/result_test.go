package types

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResult_Creation(t *testing.T) {
	t.Run("Ok creates successful result", func(t *testing.T) {
		result := Ok(42)

		assert.True(t, result.IsOk())
		assert.False(t, result.IsErr())
		assert.Equal(t, 42, result.Unwrap())
		assert.NoError(t, result.Error())
	})

	t.Run("Err creates failed result", func(t *testing.T) {
		err := errors.New("test error")
		result := Err[int](err)

		assert.False(t, result.IsOk())
		assert.True(t, result.IsErr())
		assert.Equal(t, err, result.Error())
	})
}

func TestResult_Unwrap(t *testing.T) {
	t.Run("Unwrap returns value for Ok result", func(t *testing.T) {
		result := Ok("hello")
		assert.Equal(t, "hello", result.Unwrap())
	})

	t.Run("Unwrap panics for Err result", func(t *testing.T) {
		result := Err[string](errors.New("test error"))

		assert.Panics(t, func() {
			result.Unwrap()
		})
	})
}

func TestResult_UnwrapOr(t *testing.T) {
	tests := []struct {
		name         string
		result       Result[int]
		defaultValue int
		expected     int
	}{
		{
			name:         "Ok result returns value",
			result:       Ok(42),
			defaultValue: 0,
			expected:     42,
		},
		{
			name:         "Err result returns default",
			result:       Err[int](errors.New("error")),
			defaultValue: 99,
			expected:     99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.result.UnwrapOr(tt.defaultValue)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestResult_UnwrapOrElse(t *testing.T) {
	t.Run("Ok result returns value", func(t *testing.T) {
		result := Ok(42)
		actual := result.UnwrapOrElse(func(error) int { return 0 })
		assert.Equal(t, 42, actual)
	})

	t.Run("Err result calls function", func(t *testing.T) {
		err := errors.New("test error")
		result := Err[int](err)

		var capturedErr error
		actual := result.UnwrapOrElse(func(e error) int {
			capturedErr = e
			return 99
		})

		assert.Equal(t, 99, actual)
		assert.Equal(t, err, capturedErr)
	})
}

func TestResult_Value(t *testing.T) {
	t.Run("Ok result returns value and nil error", func(t *testing.T) {
		result := Ok("test")
		value, err := result.Value()

		assert.Equal(t, "test", value)
		assert.NoError(t, err)
	})

	t.Run("Err result returns zero value and error", func(t *testing.T) {
		testErr := errors.New("test error")
		result := Err[string](testErr)
		value, err := result.Value()

		assert.Equal(t, "", value)
		assert.Equal(t, testErr, err)
	})
}

func TestResult_Map(t *testing.T) {
	t.Run("Map transforms Ok value", func(t *testing.T) {
		result := Ok(5)
		mapped := Map(result, func(x int) string {
			return fmt.Sprintf("value: %d", x)
		})

		assert.True(t, mapped.IsOk())
		assert.Equal(t, "value: 5", mapped.Unwrap())
	})

	t.Run("Map preserves Err", func(t *testing.T) {
		err := errors.New("original error")
		result := Err[int](err)
		mapped := Map(result, func(x int) string {
			return fmt.Sprintf("value: %d", x)
		})

		assert.True(t, mapped.IsErr())
		assert.Equal(t, err, mapped.Error())
	})
}

func TestResult_MapErr(t *testing.T) {
	t.Run("MapErr preserves Ok value", func(t *testing.T) {
		result := Ok(42)
		mapped := result.MapErr(func(err error) error {
			return fmt.Errorf("wrapped: %w", err)
		})

		assert.True(t, mapped.IsOk())
		assert.Equal(t, 42, mapped.Unwrap())
	})

	t.Run("MapErr transforms error", func(t *testing.T) {
		originalErr := errors.New("original")
		result := Err[int](originalErr)
		mapped := result.MapErr(func(err error) error {
			return fmt.Errorf("wrapped: %w", err)
		})

		assert.True(t, mapped.IsErr())
		assert.Contains(t, mapped.Error().Error(), "wrapped: original")
	})
}

func TestResult_FlatMap(t *testing.T) {
	t.Run("FlatMap chains Ok results", func(t *testing.T) {
		result := Ok(5)
		chained := FlatMap(result, func(x int) Result[string] {
			if x > 0 {
				return Ok(fmt.Sprintf("positive: %d", x))
			}
			return Err[string](errors.New("not positive"))
		})

		assert.True(t, chained.IsOk())
		assert.Equal(t, "positive: 5", chained.Unwrap())
	})

	t.Run("FlatMap handles function returning Err", func(t *testing.T) {
		result := Ok(-5)
		chained := FlatMap(result, func(x int) Result[string] {
			if x > 0 {
				return Ok(fmt.Sprintf("positive: %d", x))
			}
			return Err[string](errors.New("not positive"))
		})

		assert.True(t, chained.IsErr())
		assert.Equal(t, "not positive", chained.Error().Error())
	})

	t.Run("FlatMap preserves original Err", func(t *testing.T) {
		originalErr := errors.New("original error")
		result := Err[int](originalErr)
		chained := FlatMap(result, func(_ int) Result[string] {
			return Ok("never reached")
		})

		assert.True(t, chained.IsErr())
		assert.Equal(t, originalErr, chained.Error())
	})
}

func TestResult_AndThen(t *testing.T) {
	t.Run("AndThen is alias for FlatMap", func(t *testing.T) {
		result := Ok(10)

		flatMapped := FlatMap(result, func(x int) Result[int] {
			return Ok(x * 2)
		})

		andThen := AndThen(result, func(x int) Result[int] {
			return Ok(x * 2)
		})

		assert.Equal(t, flatMapped.Unwrap(), andThen.Unwrap())
	})
}

func TestResult_Or(t *testing.T) {
	t.Run("Or returns first result if Ok", func(t *testing.T) {
		first := Ok(42)
		second := Ok(99)
		result := first.Or(second)

		assert.Equal(t, 42, result.Unwrap())
	})

	t.Run("Or returns second result if first is Err", func(t *testing.T) {
		first := Err[int](errors.New("first error"))
		second := Ok(99)
		result := first.Or(second)

		assert.Equal(t, 99, result.Unwrap())
	})

	t.Run("Or returns second result even if both are Err", func(t *testing.T) {
		firstErr := errors.New("first error")
		secondErr := errors.New("second error")
		first := Err[int](firstErr)
		second := Err[int](secondErr)
		result := first.Or(second)

		assert.True(t, result.IsErr())
		assert.Equal(t, secondErr, result.Error())
	})
}

func TestResult_OrElse(t *testing.T) {
	t.Run("OrElse returns original if Ok", func(t *testing.T) {
		result := Ok(42)
		orElse := result.OrElse(func(error) Result[int] {
			return Ok(99)
		})

		assert.Equal(t, 42, orElse.Unwrap())
	})

	t.Run("OrElse calls function if Err", func(t *testing.T) {
		originalErr := errors.New("original")
		result := Err[int](originalErr)

		var capturedErr error
		orElse := result.OrElse(func(err error) Result[int] {
			capturedErr = err
			return Ok(99)
		})

		assert.Equal(t, 99, orElse.Unwrap())
		assert.Equal(t, originalErr, capturedErr)
	})
}

func TestResult_String(t *testing.T) {
	t.Run("Ok result string representation", func(t *testing.T) {
		result := Ok(42)
		assert.Equal(t, "Ok(42)", result.String())
	})

	t.Run("Err result string representation", func(t *testing.T) {
		err := errors.New("test error")
		result := Err[int](err)
		assert.Equal(t, "Err(test error)", result.String())
	})
}

func TestResult_Match(t *testing.T) {
	t.Run("Match calls onOk for Ok result", func(t *testing.T) {
		result := Ok(42)
		var called string
		var value int

		result.Match(
			func(v int) {
				called = "ok"
				value = v
			},
			func(error) {
				called = "err"
			},
		)

		assert.Equal(t, "ok", called)
		assert.Equal(t, 42, value)
	})

	t.Run("Match calls onErr for Err result", func(t *testing.T) {
		testErr := errors.New("test error")
		result := Err[int](testErr)
		var called string
		var capturedErr error

		result.Match(
			func(int) {
				called = "ok"
			},
			func(err error) {
				called = "err"
				capturedErr = err
			},
		)

		assert.Equal(t, "err", called)
		assert.Equal(t, testErr, capturedErr)
	})
}

func TestTryFrom(t *testing.T) {
	t.Run("TryFrom creates Ok from nil error", func(t *testing.T) {
		result := TryFrom(42, nil)
		assert.True(t, result.IsOk())
		assert.Equal(t, 42, result.Unwrap())
	})

	t.Run("TryFrom creates Err from non-nil error", func(t *testing.T) {
		err := errors.New("test error")
		result := TryFrom(42, err)
		assert.True(t, result.IsErr())
		assert.Equal(t, err, result.Error())
	})
}

func TestCollect(t *testing.T) {
	t.Run("Collect all Ok results", func(t *testing.T) {
		results := []Result[int]{
			Ok(1),
			Ok(2),
			Ok(3),
		}

		collected := Collect(results)
		assert.True(t, collected.IsOk())
		assert.Equal(t, []int{1, 2, 3}, collected.Unwrap())
	})

	t.Run("Collect returns first error", func(t *testing.T) {
		firstErr := errors.New("first error")
		secondErr := errors.New("second error")
		results := []Result[int]{
			Ok(1),
			Err[int](firstErr),
			Ok(3),
			Err[int](secondErr),
		}

		collected := Collect(results)
		assert.True(t, collected.IsErr())
		assert.Equal(t, firstErr, collected.Error())
	})

	t.Run("Collect empty slice", func(t *testing.T) {
		results := []Result[int]{}
		collected := Collect(results)
		assert.True(t, collected.IsOk())
		assert.Equal(t, []int{}, collected.Unwrap())
	})
}

func TestPartition(t *testing.T) {
	t.Run("Partition mixed results", func(t *testing.T) {
		err1 := errors.New("error 1")
		err2 := errors.New("error 2")
		results := []Result[int]{
			Ok(1),
			Err[int](err1),
			Ok(3),
			Err[int](err2),
			Ok(5),
		}

		values, errors := Partition(results)

		assert.Equal(t, []int{1, 3, 5}, values)
		assert.Equal(t, []error{err1, err2}, errors)
	})

	t.Run("Partition all Ok results", func(t *testing.T) {
		results := []Result[int]{Ok(1), Ok(2), Ok(3)}
		values, errors := Partition(results)

		assert.Equal(t, []int{1, 2, 3}, values)
		assert.Empty(t, errors)
	})

	t.Run("Partition all Err results", func(t *testing.T) {
		err1 := errors.New("error 1")
		err2 := errors.New("error 2")
		results := []Result[int]{Err[int](err1), Err[int](err2)}
		values, errors := Partition(results)

		assert.Empty(t, values)
		assert.Equal(t, []error{err1, err2}, errors)
	})

	t.Run("Partition empty slice", func(t *testing.T) {
		results := []Result[int]{}
		values, errors := Partition(results)

		assert.Empty(t, values)
		assert.Empty(t, errors)
	})
}

// Property-based testing helpers
func TestResult_Properties(t *testing.T) {
	t.Run("Map preserves Ok-ness", func(t *testing.T) {
		testCases := []Result[int]{
			Ok(1), Ok(42), Ok(-5), Ok(0),
		}

		for _, result := range testCases {
			mapped := Map(result, func(x int) int { return x * 2 })
			assert.Equal(t, result.IsOk(), mapped.IsOk())
		}
	})

	t.Run("Map preserves Err-ness", func(t *testing.T) {
		testCases := []Result[int]{
			Err[int](errors.New("error 1")),
			Err[int](errors.New("error 2")),
		}

		for _, result := range testCases {
			mapped := Map(result, func(x int) int { return x * 2 })
			assert.Equal(t, result.IsErr(), mapped.IsErr())
			assert.Equal(t, result.Error(), mapped.Error())
		}
	})

	t.Run("FlatMap associativity", func(t *testing.T) {
		// (m.flatMap(f)).flatMap(g) == m.flatMap(x => f(x).flatMap(g))
		result := Ok(5)

		f := func(x int) Result[int] { return Ok(x * 2) }
		g := func(x int) Result[int] { return Ok(x + 1) }

		left := FlatMap(FlatMap(result, f), g)
		right := FlatMap(result, func(x int) Result[int] {
			return FlatMap(f(x), g)
		})

		assert.Equal(t, left.Unwrap(), right.Unwrap())
	})
}

// Benchmark tests for performance
func BenchmarkResult_Map(b *testing.B) {
	result := Ok(42)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Map(result, func(x int) int { return x * 2 })
	}
}

func BenchmarkResult_FlatMap(b *testing.B) {
	result := Ok(42)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FlatMap(result, func(x int) Result[int] { return Ok(x * 2) })
	}
}

func BenchmarkCollect(b *testing.B) {
	results := make([]Result[int], 100)
	for i := 0; i < 100; i++ {
		results[i] = Ok(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Collect(results)
	}
}

// Example usage for documentation
func ExampleMap() {
	result := Ok(5)
	doubled := Map(result, func(x int) int { return x * 2 })

	fmt.Println(doubled.Unwrap())
	// Output: 10
}

func ExampleFlatMap() {
	divide := func(a, b int) Result[int] {
		if b == 0 {
			return Err[int](errors.New("division by zero"))
		}
		return Ok(a / b)
	}

	result := Ok(10)
	divided := FlatMap(result, func(x int) Result[int] {
		return divide(x, 2)
	})

	fmt.Println(divided.Unwrap())
	// Output: 5
}

func ExampleCollect() {
	results := []Result[int]{
		Ok(1),
		Ok(2),
		Ok(3),
	}

	collected := Collect(results)
	fmt.Println(collected.Unwrap())
	// Output: [1 2 3]
}
