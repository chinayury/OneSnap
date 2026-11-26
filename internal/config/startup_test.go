package config

import (
	"crypto/rand"
	"fmt"
	"os"
	"testing"

	"golang.org/x/sys/windows/registry"
)

func TestStartupRegistry(t *testing.T) {
	path := `Software\OneSnapTest-` + rand.Text()
	t.Cleanup(func() {
		err := registry.DeleteKey(registry.CURRENT_USER, path)
		if err != nil && err != registry.ErrNotExist {
			t.Errorf("clean up temporary key: %v", err)
		}
	})

	checkEnabled := func(want bool) {
		t.Helper()
		got, err := isStartupEnabled(path)
		if err != nil || got != want {
			t.Fatalf("startup enabled = %v, %v; want %v, nil", got, err, want)
		}
	}
	checkEnabled(false)
	if err := disableStartup(path); err != nil {
		t.Fatal(err)
	}
	if err := enableStartup(path); err != nil {
		t.Fatal(err)
	}
	checkEnabled(true)

	key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := key.GetStringValue(appName)
	key.Close()
	if err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if want := fmt.Sprintf(`"%s"`, exe); got != want {
		t.Fatalf("startup command = %q, want %q", got, want)
	}

	for i := 0; i < 2; i++ {
		if err := disableStartup(path); err != nil {
			t.Fatal(err)
		}
		checkEnabled(false)
	}
}
