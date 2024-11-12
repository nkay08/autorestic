package internal

import "testing"

func TestTopologicalSorting(t *testing.T) {
	t.Run("simple string well defined", func(t *testing.T) {
		adjList := map[string][]string{
			"a": {},
			"b": {"c"},
			"c": {"a"},
		}
		result, error := TopologicalSort(adjList, true)
		assertEqual(t, error, nil)
		assertSliceEqual(t, result, []string{"a", "c", "b"})
	})

	t.Run("simple int well defined", func(t *testing.T) {
		adjList := map[int][]int{
			1: {},
			2: {3},
			3: {1},
		}
		result, error := TopologicalSort(adjList, true)
		assertEqual(t, error, nil)
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
		result, error := TopologicalSort(adjList, true)
		assertEqual(t, error, nil)
		assertSliceEqual(t, result, []string{"a", "c", "b"})
	})

	t.Run("cyclic dependency", func(t *testing.T) {
		adjList := map[string][]string{
			"a": {},
			"b": {"c"},
			"c": {"b"},
		}
		_, error := TopologicalSort(adjList, true)
		if error == nil {
			t.Fatalf("No cyclic dependency found, but test data includes it!")
		}
	})

	t.Run("string empty edges", func(t *testing.T) {
		// We cannot make an assumption about the order of iteration of the map!
		// But it should not produce an error
		adjList := map[string][]string{
			"a": {},
			"b": {},
			"c": {},
		}
		result, error := TopologicalSort(adjList, false)
		assertEqual(t, error, nil)
		assertEqual(t, len(result), 3)
	})

	t.Run("string reverse original order empty edges", func(t *testing.T) {
		// We cannot make an assumption about the order of iteration of the map!
		// But it should not produce an error
		adjList := map[string][]string{
			"a": {},
			"b": {},
			"c": {},
		}
		result, error := TopologicalSort(adjList, true)
		assertEqual(t, error, nil)
		assertEqual(t, len(result), 3)
	})

	t.Run("complex order well defined", func(t *testing.T) {
		adjList := map[string][]string{
			"a": {"e"},
			"b": {"c"},
			"c": {},
			"d": {"c", "a"},
			"e": {"b", "c"},
		}
		result, error := TopologicalSort(adjList, true)
		assertEqual(t, error, nil)
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
		result, error := TopologicalSort(adjList, true)
		assertEqual(t, error, nil)
		assertSliceEqual(t, result, []string{"c", "b", "e", "a", "d", "f", "g"})
	})
}
