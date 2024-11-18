package internal

import (
	"testing"
	"time"

	"github.com/cupcakearmy/autorestic/internal/lock"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestGetType(t *testing.T) {

	t.Run("TypeLocal", func(t *testing.T) {
		l := Location{
			Type: "local",
		}
		result, err := l.getType()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		assertEqual(t, result, TypeLocal)
	})

	t.Run("TypeVolume", func(t *testing.T) {
		l := Location{
			Type: "volume",
		}
		result, err := l.getType()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		assertEqual(t, result, TypeVolume)
	})

	t.Run("Empty type", func(t *testing.T) {
		l := Location{
			Type: "",
		}
		result, err := l.getType()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		assertEqual(t, result, TypeLocal)
	})

	t.Run("Invalid type", func(t *testing.T) {
		l := Location{
			Type: "foo",
		}
		_, err := l.getType()
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestBuildTag(t *testing.T) {
	result := buildTag("foo", "bar")
	expected := "ar:foo:bar"
	assertEqual(t, result, expected)
}

func TestGetLocationTags(t *testing.T) {
	l := Location{
		name: "foo",
	}
	result := l.getLocationTags()
	expected := "ar:location:foo"
	assertEqual(t, result, expected)
}

func TestHasBackend(t *testing.T) {
	t.Run("backend present", func(t *testing.T) {
		l := Location{
			name: "foo",
			To:   []string{"foo", "bar"},
		}
		result := l.hasBackend("foo")
		assertEqual(t, result, true)
	})

	t.Run("backend absent", func(t *testing.T) {
		l := Location{
			name: "foo",
			To:   []string{"bar", "baz"},
		}
		result := l.hasBackend("foo")
		assertEqual(t, result, false)
	})
}

func TestBuildRestoreCommand(t *testing.T) {
	l := Location{
		name: "foo",
	}
	result := buildRestoreCommand(l, "to", "snapshot", []string{"options"})
	expected := []string{"restore", "--target", "to", "--tag", "ar:location:foo", "snapshot", "options"}
	assertSliceEqual(t, result, expected)
}

func TestCron(t *testing.T) {
	now := time.Now()

	loc := Location{
		name: "a",
	}

	t.Run("check empty", func(t *testing.T) {
		loc.Cron = ""
		runCron, err := loc.CheckCron()
		assert.Empty(t, err)
		assert.False(t, runCron)
	})

	t.Run("check wrong cron format ", func(t *testing.T) {
		loc.Cron = "xyada"
		runCron, err := loc.CheckCron()
		assert.NotEmpty(t, err)
		assert.False(t, runCron)
	})

	// create virtual file
	fs := new(afero.MemMapFs)
	_, err := afero.TempFile(fs, "", ".autorestic.yml")

	assert.Empty(t, err)
	viper.SetConfigFile(".")

	t.Run("check due ", func(t *testing.T) {
		// start of epoch 1970-01-01
		lock.SetCron(loc.name, 0)
		// every minute
		loc.Cron = "* * * * *"
		runCron, err := loc.CheckCron()
		assert.Empty(t, err)
		assert.True(t, runCron)
	})

	t.Run("check not due ", func(t *testing.T) {
		lock.SetCron(loc.name, now.Unix())
		// every 7 days
		loc.Cron = "0 0 * * 0"
		runCron, err := loc.CheckCron()
		assert.Empty(t, err)
		assert.False(t, runCron)
	})
}
