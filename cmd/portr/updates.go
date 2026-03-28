package main

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/Masterminds/semver"
	"github.com/amalshaji/portr/internal/client/config"
	"github.com/go-resty/resty/v2"
)

var UpdatesFilePath = config.DefaultConfigDir + "/updates.json"

type GithubRelease struct {
	TagName string `json:"tag_name"`
}

type UpdateInfo struct {
	CheckedAt time.Time `json:"checked_at"`
	Version   string    `json:"version"`
}

func createUpdatesFileIfNotExists() error {
	if _, err := os.Stat(UpdatesFilePath); err != nil {
		if os.IsNotExist(err) {
			f, err := os.Create(UpdatesFilePath)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := f.WriteString("{}"); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func getUpdateState() (UpdateInfo, error) {
	data, err := os.ReadFile(UpdatesFilePath)
	if err != nil {
		return UpdateInfo{}, err
	}

	var updateInfo UpdateInfo
	err = json.Unmarshal(data, &updateInfo)
	if err != nil {
		return UpdateInfo{}, err
	}

	return updateInfo, nil
}

func checkForUpdates() error {
	if err := createUpdatesFileIfNotExists(); err != nil {
		return err
	}

	updateInfo, err := getUpdateState()
	if err != nil {
		return err
	}

	if updateInfo.Version == "" {
		return getLatestRelease()
	}

	currentVersion, err := semver.NewVersion(version)
	if err != nil {
		return err
	}

	lastCheckedVersion, err := semver.NewVersion(updateInfo.Version)
	if err != nil {
		return err
	}

	if updateInfo.CheckedAt.Before(time.Now().Add(-24*time.Hour)) && currentVersion.Equal(lastCheckedVersion) {
		return getLatestRelease()
	}

	return nil
}

func getLatestRelease() error {
	if err := createUpdatesFileIfNotExists(); err != nil {
		return err
	}

	var lastCheck UpdateInfo
	data, err := os.ReadFile(UpdatesFilePath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &lastCheck)
	if err != nil {
		return err
	}

	if time.Since(lastCheck.CheckedAt) < 24*time.Hour {
		return nil
	}

	client := resty.New()

	var release GithubRelease
	resp, err := client.R().SetResult(&release).Get("https://api.github.com/repos/amalshaji/portr/releases/latest")
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		return err
	}

	updateInfo := UpdateInfo{
		CheckedAt: time.Now().UTC(),
		Version:   release.TagName,
	}

	data, err = json.Marshal(updateInfo)
	if err != nil {
		return err
	}

	if err := os.WriteFile(UpdatesFilePath, data, 0644); err != nil {
		return err
	}

	return nil
}

func getVersionToUpdate() (string, error) {
	updateInfo, err := getUpdateState()
	if err != nil {
		return "", err
	}

	if updateInfo.Version == "" {
		return "", nil
	}

	currentVersion, err := semver.NewVersion(version)
	if err != nil {
		return "", err
	}

	latestVersion, err := semver.NewVersion(updateInfo.Version)
	if err != nil {
		return "", err
	}

	if currentVersion.LessThan(latestVersion) {
		return updateInfo.Version, nil
	}

	return "", nil
}
