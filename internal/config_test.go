package internal

import (
	"path"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLocationSorting(t *testing.T) {

	t.Run("test sort, well defined", func(t *testing.T) {
		locationsMap := map[string]Location{
			"c": {
				name:      "c",
				DependsOn: []string{"b", "a"},
			},
			"a": {
				name: "a",
			},
			"b": {
				name:      "b",
				DependsOn: []string{"a"},
			},
			"d": {
				name:      "d",
				DependsOn: []string{"c"},
			},
			"e": {
				name:      "e",
				DependsOn: []string{"b", "d", "a"},
			},
		}

		result, error := SortLocationsTopologicalFromMap(locationsMap)
		assertEqual(t, error, nil)
		assertSliceEqual(t, result, []string{"a", "b", "c", "d", "e"})
	})

	t.Run("test empty dependency", func(t *testing.T) {

		locationsMap := map[string]Location{
			"9": {
				name: "9",
			},
			"1": {
				name:      "1",
				DependsOn: []string{"9"},
			},
			"5": {
				name:      "5",
				DependsOn: []string{"1"},
			},
		}

		result, error := SortLocationsTopologicalFromMap(locationsMap)
		assertEqual(t, error, nil)
		assertSliceEqual(t, result, []string{"9", "1", "5"})
	})
}

func TestGetSelectedLocations(t *testing.T) {
	configInitial := Config{
		Locations: map[string]Location{
			"a": {
				name: "a",
			},
			"b": {
				name: "b",
			},
			"c": {
				name: "c",
			},
		},
	}
	config = &configInitial

	t.Run("test all Locations", func(t *testing.T) {
		_, ok := GetLocation("a")
		if !ok {
			t.Fail()
		}

		cmd := cobra.Command{}
		AddFlagsToCommand(&cmd, false)
		cmd.SetArgs([]string{"--all"})
		cmd.Execute()

		all, _ := cmd.Flags().GetBool("all")
		assertEqual(t, all, true)

		locList, err2 := GetAllOrSelected(&cmd, false)
		assertEqual(t, err2, nil)
		assertEqual(t, len(locList), 3)
		assert.ElementsMatch(t, locList, []string{"a", "b", "c"})
	})

	t.Run("test no location selected", func(t *testing.T) {
		cmd := cobra.Command{}
		AddFlagsToCommand(&cmd, false)
		cmd.SetArgs([]string{})
		cmd.Execute()

		strlist, _ := cmd.Flags().GetStringSlice("location")
		assert.Empty(t, strlist)
		// assert.NotEmpty(t, err)

		all, _ := cmd.Flags().GetBool("all")
		assert.Equal(t, all, false)

		locs, _ := GetAllOrSelected(&cmd, false)
		assert.Empty(t, locs)
	})

	t.Run("test select some locations", func(t *testing.T) {
		cmd := cobra.Command{}
		AddFlagsToCommand(&cmd, false)
		cmd.SetArgs([]string{"-l", "a"})
		cmd.Execute()

		locs, err2 := GetAllOrSelected(&cmd, false)
		assert.Equal(t, err2, nil)
		assert.Contains(t, locs, "a")

		cmd = cobra.Command{}
		AddFlagsToCommand(&cmd, false)
		cmd.SetArgs([]string{"-l", "a", "-l", "c"})
		cmd.Execute()
		locs, err2 = GetAllOrSelected(&cmd, false)
		assert.Equal(t, err2, nil)
		assert.Contains(t, locs, "a")
		assert.Contains(t, locs, "c")
	})

	t.Run("test select not existing", func(t *testing.T) {
		cmd := cobra.Command{}
		AddFlagsToCommand(&cmd, false)
		cmd.SetArgs([]string{"-l", "d"})
		cmd.Execute()

		_, err := GetAllOrSelected(&cmd, false)
		assert.NotEmpty(t, err)
	})

	t.Run("test sorting", func(t *testing.T) {
		configTopological := Config{
			Locations: map[string]Location{
				"a": {
					name:      "a",
					DependsOn: []string{"c"},
				},
				"b": {
					name:      "b",
					DependsOn: []string{"a"},
				},
				"c": {
					name: "c",
				},
			},
		}

		config = &configTopological

		cmd := cobra.Command{}
		AddFlagsToCommand(&cmd, false)
		cmd.SetArgs([]string{"--all"})
		cmd.Execute()

		sortedLocStrings, err := GetAllOrSelected(&cmd, false)
		assert.Empty(t, err)
		assertSliceEqual(t, sortedLocStrings, []string{"c", "a", "b"})

		config = &configInitial
	})
}

func TestOptionToString(t *testing.T) {
	t.Run("no prefix", func(t *testing.T) {
		opt := "test"
		result := optionToString(opt)
		assertEqual(t, result, "--test")
	})

	t.Run("single prefix", func(t *testing.T) {
		opt := "-test"
		result := optionToString(opt)
		assertEqual(t, result, "-test")
	})

	t.Run("double prefix", func(t *testing.T) {
		opt := "--test"
		result := optionToString(opt)
		assertEqual(t, result, "--test")
	})
}

func TestAppendOneOptionToSlice(t *testing.T) {
	t.Run("string flag", func(t *testing.T) {
		result := []string{}
		optionMap := OptionMap{"string-flag": []interface{}{"/root"}}

		appendOptionsToSlice(&result, optionMap)
		expected := []string{
			"--string-flag", "/root",
		}
		assertSliceEqual(t, result, expected)
	})

	t.Run("bool flag", func(t *testing.T) {
		result := []string{}
		optionMap := OptionMap{"boolean-flag": []interface{}{true}}

		appendOptionsToSlice(&result, optionMap)
		expected := []string{
			"--boolean-flag",
		}
		assertSliceEqual(t, result, expected)
	})

	t.Run("int flag", func(t *testing.T) {
		result := []string{}
		optionMap := OptionMap{"int-flag": []interface{}{123}}

		appendOptionsToSlice(&result, optionMap)
		expected := []string{
			"--int-flag", "123",
		}
		assertSliceEqual(t, result, expected)
	})
}

func TestAppendMultipleOptionsToSlice(t *testing.T) {
	result := []string{}
	optionMap := OptionMap{
		"string-flag": []interface{}{"/root"},
		"int-flag":    []interface{}{123},
	}

	appendOptionsToSlice(&result, optionMap)
	expected := []string{
		"--string-flag", "/root",
		"--int-flag", "123",
	}
	if len(result) != len(expected) {
		t.Errorf("got length %d, want length %d", len(result), len(expected))
	}

	// checks that expected option comes after flag, regardless of key order in map
	for i, v := range expected {
		v = strings.TrimPrefix(v, "--")

		if value, ok := optionMap[v]; ok {
			if val, ok := value[0].(int); ok {
				if expected[i+1] != strconv.Itoa(val) {
					t.Errorf("Flags and options order are mismatched. got %v, want %v", result, expected)
				}
			}
		}
	}
}

func TestAppendOptionWithMultipleValuesToSlice(t *testing.T) {
	result := []string{}
	optionMap := OptionMap{
		"string-flag": []interface{}{"/root", "/bin"},
	}

	appendOptionsToSlice(&result, optionMap)
	expected := []string{
		"--string-flag", "/root",
		"--string-flag", "/bin",
	}
	assertSliceEqual(t, result, expected)
}

func TestGetOptionsOneKey(t *testing.T) {
	optionMap := OptionMap{
		"string-flag": []interface{}{"/root"},
	}
	options := Options{"backend": optionMap}
	keys := []string{"backend"}

	result := getOptions(options, keys)
	expected := []string{
		"--string-flag", "/root",
	}
	assertSliceEqual(t, result, expected)
}

func TestGetOptionsMultipleKeys(t *testing.T) {
	firstOptionMap := OptionMap{
		"string-flag": []interface{}{"/root"},
	}
	secondOptionMap := OptionMap{
		"boolean-flag": []interface{}{true},
		"int-flag":     []interface{}{123},
	}
	options := Options{
		"all":    firstOptionMap,
		"forget": secondOptionMap,
	}
	keys := []string{"all", "forget"}

	result := getOptions(options, keys)
	expected := []string{
		"--string-flag", "/root",
		"--boolean-flag",
		"--int-flag", "123",
	}
	reflect.DeepEqual(result, expected)
}

func TestSaveConfigProducesReadableConfig(t *testing.T) {
	workDir := t.TempDir()
	viper.SetConfigFile(path.Join(workDir, ".autorestic.yml"))

	// Required to appease the config reader
	viper.Set("version", 2)

	c := Config{
		Version: "2",
		Locations: map[string]Location{
			"test": {
				Type: "local",
				name: "test",
				From: []string{"in-dir"},
				To:   []string{"test"},
				// ForgetOption & ConfigOption have previously marshalled in a way that
				// can't get read correctly
				ForgetOption: "foo",
				CopyOption:   map[string][]string{"foo": {"bar"}},
			},
		},
		Backends: map[string]Backend{
			"test": {
				name: "test",
				Type: "local",
				Path: "backup-target",
				Key:  "supersecret",
			},
		},
	}

	err := c.SaveConfig()
	assert.NoError(t, err)

	// Ensure we the config reading logic actually runs
	config = nil
	once = sync.Once{}
	readConfig := GetConfig()
	assert.NotNil(t, readConfig)
	assert.Equal(t, c, *readConfig)
}

func assertEqual[T comparable](t testing.TB, result, expected T) {
	t.Helper()

	if result != expected {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func assertSliceEqual(t testing.TB, result, expected []string) {
	t.Helper()

	if len(result) != len(expected) {
		t.Errorf("got length %d, want length %d", len(result), len(expected))
	}

	for i := range result {
		assertEqual(t, result[i], expected[i])
	}
}
