package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTopologicalSorting(t *testing.T) {
	t.Run("simple string well defined", func(t *testing.T) {
		adjList := map[string][]string{
			"a": {},
			"b": {"c"},
			"c": {"a"},
		}
		result, err := TopologicalSort(adjList, true)
		assert.Empty(t, err)
		assertSliceEqual(t, result, []string{"a", "c", "b"})
	})

	t.Run("simple int well defined", func(t *testing.T) {
		adjList := map[int][]int{
			1: {},
			2: {3},
			3: {1},
		}
		result, err := TopologicalSort(adjList, true)
		assert.Empty(t, err)
		expected := []int{1, 3, 2}
		for i, elem := range result {
			if elem != expected[i] {
				t.Fail()
			}
		}
	})

	t.Run("simple string non-unique well defined", func(t *testing.T) {
		adjList := map[string][]string{
			"a": {},
			"b": {"c", "c", "c"},
			"c": {"a", "a"},
		}
		result, err := TopologicalSort(adjList, true)
		assert.Empty(t, err)
		assertSliceEqual(t, result, []string{"a", "c", "b"})
	})

	t.Run("cyclic dependency", func(t *testing.T) {
		adjList := map[string][]string{
			"a": {},
			"b": {"c"},
			"c": {"b"},
		}
		_, err := TopologicalSort(adjList, true)
		assert.NotEmpty(t, err)
	})

	t.Run("string empty edges", func(t *testing.T) {
		// We cannot make an assumption about the order of iteration of the map!
		// But it should not produce an error
		adjList := map[string][]string{
			"a": {},
			"b": {},
			"c": {},
		}
		result, err := TopologicalSort(adjList, false)
		assert.Empty(t, err)
		assert.Equal(t, 3, len(result))
	})

	t.Run("string reverse original order empty edges", func(t *testing.T) {
		// We cannot make an assumption about the order of iteration of the map!
		// But it should not produce an error
		adjList := map[string][]string{
			"a": {},
			"b": {},
			"c": {},
		}
		result, err := TopologicalSort(adjList, true)
		assert.Empty(t, err)
		assert.Equal(t, 3, len(result))
	})

	t.Run("complex order well defined", func(t *testing.T) {
		adjList := map[string][]string{
			"a": {"e"},
			"b": {"c"},
			"c": {},
			"d": {"c", "a"},
			"e": {"b", "c"},
		}
		result, err := TopologicalSort(adjList, true)
		assert.Empty(t, err)
		assertSliceEqual(t, result, []string{"c", "b", "e", "a", "d"})
	})

	t.Run("complex order well defined2", func(t *testing.T) {
		adjList := map[string][]string{
			"a": {"e"},
			"b": {"c"},
			"c": {},
			"d": {"c", "a"},
			"e": {"b", "c"},
			"f": {"a", "c", "e", "b", "d"},
			"g": {"c", "f"},
		}
		result, err := TopologicalSort(adjList, true)
		assert.Empty(t, err)
		assertSliceEqual(t, result, []string{"c", "b", "e", "a", "d", "f", "g"})
	})
}
