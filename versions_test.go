package main

import (
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestGetLocalVersionsLogic(t *testing.T) {
	// Setup temp dir simulating AutoZapret
	tmpDir, err := os.MkdirTemp("", "autozapret_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Mock environment variable?
	// GetLocalVersions uses os.Getenv("LOCALAPPDATA"). We can't easily mock that in parallel tests without side effects.
	// But we can extract the logic or just set env for this test process (since we run `go test`).
	originalEnv := os.Getenv("LOCALAPPDATA")
	// We need to set LOCALAPPDATA such that `filepath.Join(localAppData, "AutoZapret")` resolves to our tmpDir.
	// So set LOCALAPPDATA to parent of tmpDir/AutoZapret.
	// Wait, we can just make tmpDir/AutoZapret.

	// Create "AutoZapret" inside tmpDir
	autoZapretDir := filepath.Join(tmpDir, "AutoZapret")
	if err := os.Mkdir(autoZapretDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Set env
	os.Setenv("LOCALAPPDATA", tmpDir)
	defer os.Setenv("LOCALAPPDATA", originalEnv)

	// Create dummy folders
	folders := []string{
		"zapret-discord-youtube-1.9.3",
		"zapret-v1.8.5-BF-v3.2",
		"zapret-discord-youtube-1.9.4",
		"random-folder",
		"file.txt",
	}

	for _, f := range folders {
		path := filepath.Join(autoZapretDir, f)
		if f == "file.txt" {
			os.WriteFile(path, []byte("test"), 0644)
		} else {
			os.Mkdir(path, 0755)
		}
	}

	versions, err := GetLocalVersions()
	if err != nil {
		t.Fatal(err)
	}

	// Expecting:
	// 1.9.3 (official)
	// 1.9.4 (official)
	// zapret-v1.8.5-BF-v3.2 (custom)
	// random-folder (IGNORED because no "zapret-" prefix)

	expectedCount := 3
	if len(versions) != expectedCount {
		t.Errorf("Expected %d versions, got %d", expectedCount, len(versions))
		for _, v := range versions {
			t.Logf("Found: %s (Custom: %v)", v.Name, v.IsCustom)
		}
	}

	versionMap := make(map[string]Version)
	for _, v := range versions {
		versionMap[v.Name] = v
	}

	if v, ok := versionMap["1.9.3"]; !ok || v.IsCustom {
		t.Error("Expected 1.9.3 to be found and NOT custom")
	}
	if v, ok := versionMap["1.9.4"]; !ok || v.IsCustom {
		t.Error("Expected 1.9.4 to be found and NOT custom")
	}
	if v, ok := versionMap["zapret-v1.8.5-BF-v3.2"]; !ok || !v.IsCustom {
		t.Error("Expected zapret-v1.8.5-BF-v3.2 to be found and IS custom")
	}

	log.Println("Test passed")
}
