package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Version struct {
	Name        string
	FullPath    string // Empty if not installed
	IsInstalled bool
	IsCustom    bool
	TagName     string // For github releases
	AssetURL    string // For download
}

type GithubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadUrl string `json:"browser_download_url"`
	} `json:"assets"`
}

const ZapretRepo = "Flowseal/zapret-discord-youtube"

func getAutoZapretDir() (string, error) {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = "C:\\ProgramData"
	}
	return filepath.Join(programData, "ZapretController", "Versions"), nil
}

func GetLocalVersions() ([]Version, error) {
	dir, err := getAutoZapretDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Version{}, nil
		}
		return nil, err
	}

	var versions []Version

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()

		v := Version{
			FullPath:    filepath.Join(dir, name),
			IsInstalled: true,
			IsCustom:    true, // Default to custom
			Name:        name, // Default to full name
		}

		if strings.HasPrefix(name, "zapret-discord-youtube-") {
			v.Name = strings.TrimPrefix(name, "zapret-discord-youtube-")
			v.IsCustom = false
		} else {
			v.Name = name
			v.IsCustom = true
		}

		versions = append(versions, v)
	}
	return versions, nil
}

func GetOnlineVersions() ([]Version, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases", ZapretRepo)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch releases: %s", resp.Status)
	}

	var releases []GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	var versions []Version
	for _, r := range releases {
		// Find zip asset
		var zipUrl string
		for _, a := range r.Assets {
			if strings.HasSuffix(a.Name, ".zip") {
				zipUrl = a.BrowserDownloadUrl
				break
			}
		}

		if zipUrl == "" {
			continue // Skip releases without zip
		}

		versions = append(versions, Version{
			Name:        r.TagName,
			TagName:     r.TagName,
			AssetURL:    zipUrl,
			IsInstalled: false,
			IsCustom:    false,
		})
	}
	return versions, nil
}

// GetAllVersions merges local and online versions
func GetAllVersions() ([]Version, error) {
	local, err := GetLocalVersions()
	if err != nil {
		log.Println("Error reading local versions:", err)
		local = []Version{}
	}

	online, err := GetOnlineVersions()
	if err != nil {
		log.Println("Error fetching online versions:", err)
		// Return at least local versions
		return local, nil
	}

	versionMap := make(map[string]Version)

	for _, v := range online {
		versionMap[v.Name] = v
	}

	for _, v := range local {
		v.IsInstalled = true
		versionMap[v.Name] = v
	}

	var result []Version
	for _, v := range versionMap {
		result = append(result, v)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name > result[j].Name
	})

	return result, nil
}

func DownloadVersion(v Version) error {
	if v.AssetURL == "" {
		return fmt.Errorf("no download URL for version %s", v.Name)
	}

	log.Printf("Downloading %s from %s...\n", v.Name, v.AssetURL)

	// Temp file
	tempFile, err := os.CreateTemp("", "zapret-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Download
	resp, err := http.Get(v.AssetURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return err
	}

	// Prepare destination
	baseDir, err := getAutoZapretDir()
	if err != nil {
		return err
	}
	// Folder name: zapret-discord-youtube-<TagName>
	folderName := "zapret-discord-youtube-" + v.TagName
	destDir := filepath.Join(baseDir, folderName)

	log.Printf("Extracting to %s...\n", destDir)

	// Remove if exists (re-download)
	os.RemoveAll(destDir)

	// Unzip
	// Re-open file for reading
	tempFile.Seek(0, 0)

	// Need size
	stat, _ := tempFile.Stat()
	zipReader, err := zip.NewReader(tempFile, stat.Size())
	if err != nil {
		return err
	}

	for _, f := range zipReader.File {
		fpath := filepath.Join(destDir, f.Name)

		// Check for ZipSlip
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()
	}

	return nil
}
