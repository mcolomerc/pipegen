package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update PipeGen CLI to the latest version",
	Long:  `Automatically download and replace the PipeGen CLI with the latest release from GitHub.`,
	Run: func(cmd *cobra.Command, args []string) {
		latest, err := getLatestVersion()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to fetch latest version: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[INFO] Latest version: %s\n", latest)
		if err := selfUpdate(latest); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Update failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("[SUCCESS] PipeGen updated successfully!")
	},
}

func getLatestVersion() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/mcolomerc/pipegen/releases/latest")
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	var release struct {
		TagName string `json:"tag_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to parse release data: %v", err)
	}

	if release.TagName == "" {
		return "", fmt.Errorf("no tag_name found in response")
	}

	return release.TagName, nil
}

func selfUpdate(version string) error {
	osys := runtime.GOOS
	arch := runtime.GOARCH
	var platform, ext string

	switch osys {
	case "darwin":
		platform = "darwin"
	case "linux":
		platform = "linux"
	case "windows":
		platform = "windows"
	default:
		return fmt.Errorf("unsupported OS: %s", osys)
	}

	switch arch {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	case "386":
		arch = "386"
	default:
		return fmt.Errorf("unsupported arch: %s", arch)
	}

	switch osys {
	case "windows":
		ext = ".zip"
	default:
		ext = ".tar.gz"
	}

	url := fmt.Sprintf("https://github.com/mcolomerc/pipegen/releases/download/%s/pipegen-%s-%s%s", version, platform, arch, ext)
	fmt.Printf("[INFO] Downloading: %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	// Save to temp file
	tmpFile, err := os.CreateTemp("", "pipegen-update-*")
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := os.Remove(tmpFile.Name()); removeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove temp file: %v\n", removeErr)
		}
	}()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return err
	}

	switch ext {
	case ".tar.gz":
		if err := extractTarGz(tmpFile.Name(), platform, arch); err != nil {
			return err
		}
	case ".zip":
		if err := extractZip(tmpFile.Name(), platform, arch); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported archive format: %s", ext)
	}
	return nil
}

func extractTarGz(archive, platform, arch string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close file: %v\n", closeErr)
		}
	}()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := gz.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close gzip reader: %v\n", closeErr)
		}
	}()

	tarReader := tar.NewReader(gz)
	binName := fmt.Sprintf("pipegen-%s-%s", platform, arch)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if filepath.Base(hdr.Name) == binName {
			exePath, err := os.Executable()
			if err != nil {
				return err
			}
			out, err := os.OpenFile(exePath, os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			defer func() {
				if closeErr := out.Close(); closeErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to close output file: %v\n", closeErr)
				}
			}()

			_, err = io.Copy(out, tarReader)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("binary %s not found in archive", binName)
}

func extractZip(archive, platform, arch string) error {
	zipReader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := zipReader.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close zip reader: %v\n", closeErr)
		}
	}()

	binName := fmt.Sprintf("pipegen-%s-%s.exe", platform, arch)
	for _, file := range zipReader.File {
		if filepath.Base(file.Name) == binName {
			exePath, err := os.Executable()
			if err != nil {
				return err
			}
			out, err := os.OpenFile(exePath, os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			defer func() {
				if closeErr := out.Close(); closeErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to close output file: %v\n", closeErr)
				}
			}()

			rc, err := file.Open()
			if err != nil {
				return err
			}
			defer func() {
				if closeErr := rc.Close(); closeErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to close zip file reader: %v\n", closeErr)
				}
			}()

			_, err = io.Copy(out, rc)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("binary %s not found in archive", binName)
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
