package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/lomokwa/mc-manager/utils"
)

func DownloadLatestServerJar(destPath string, releaseVersion string) error {
	log.Printf("downloading version manifest")
	res, err := http.Get("https://launchermeta.mojang.com/mc/game/version_manifest.json")
	if err != nil {
		return fmt.Errorf("failed to fetch version manifest")
	}

	defer res.Body.Close()

	var manifest struct {
		Latest struct {
			Release  string `json:"release"`
			Snapshot string `json:"snapshot"`
		} `json:"latest"`
		Versions []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
			URL  string `json:"url"`
		}
	}

	if err := json.NewDecoder(res.Body).Decode(&manifest); err != nil {
		return fmt.Errorf("failed to decode version manifest")
	}

	var releaseId string
	if releaseVersion != "" {
		releaseId = releaseVersion
	} else {
		releaseId = manifest.Latest.Release
	}

	var versionUrl string
	for _, version := range manifest.Versions {
		if version.ID == releaseId {
			versionUrl = version.URL
			break
		}
	}

	if versionUrl == "" {
		return fmt.Errorf("latest version URL not found")
	}

	log.Printf("downloading latest version details")
	versionRes, err := http.Get(versionUrl)
	if err != nil {
		return fmt.Errorf("failed to fetch latest version details")
	}
	defer versionRes.Body.Close()

	var versionDetails struct {
		Downloads struct {
			Server struct {
				URL string `json:"url"`
			} `json:"server"`
		} `json:"downloads"`
	}

	if err := json.NewDecoder(versionRes.Body).Decode(&versionDetails); err != nil {
		return fmt.Errorf("failed to decode version details")
	}

	serverJarUrl := versionDetails.Downloads.Server.URL

	log.Printf("downloading server jar to %s", destPath)
	err = utils.DownloadFile(serverJarUrl, destPath)
	if err != nil {
		return fmt.Errorf("failed to download server.jar: %s", err)
	}

	log.Printf("server jar download complete")

	return nil
}

func PrepareServerFiles(serverDir string, createLaunchScript bool, configureProperties bool, requestProperties map[string]string) error {
	log.Printf("preparing server files in %s", serverDir)
	if err := utils.WriteFile(filepath.Join(serverDir, "eula.txt"), []byte("eula=true")); err != nil {
		return err
	}

	// Create server.properties file content.
	properties := make(map[string]string, len(DefaultServerProperties))
	for k, v := range DefaultServerProperties {
		properties[k] = v
	}

	for k, v := range requestProperties {
		properties[k] = v
	}

	var content strings.Builder
	for k, v := range properties {
		fmt.Fprintf(&content, "%s=%s\n", k, v)
	}

	propertiesContent := []byte(content.String())
	if configureProperties {
		log.Printf("writing server.properties")
		if err := utils.WriteFile(filepath.Join(serverDir, "server.properties"), propertiesContent); err != nil {
			return err
		}
	}

	if createLaunchScript {
		log.Printf("writing launch scripts")
		shellScriptPath := filepath.Join(serverDir, "start-server.sh")
		batScriptPath := filepath.Join(serverDir, "start-server.bat")

		if err := utils.WriteFile(shellScriptPath, []byte(DefaultStartServerShellScript)); err != nil {
			return fmt.Errorf("failed to write start-server.sh: %w", err)
		}

		if err := os.Chmod(shellScriptPath, 0755); err != nil {
			return fmt.Errorf("failed to set executable permission on start-server.sh: %w", err)
		}

		if err := utils.WriteFile(batScriptPath, []byte(DefaultStartServerBatchScript)); err != nil {
			return fmt.Errorf("failed to write start-server.bat: %w", err)
		}
	}

	log.Printf("server file preparation complete")

	return nil
}
