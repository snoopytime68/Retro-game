// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2021-Present The Zarf Authors

// Package utils provides generic helper functions.
package utils

import (
	"bufio"
	"crypto"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/defenseunicorns/zarf/src/pkg/message"
	"github.com/defenseunicorns/zarf/src/pkg/utils/helpers"
	"github.com/defenseunicorns/zarf/src/types"
	"github.com/otiai10/copy"
)

const (
	dotCharacter  = 46
	tmpPathPrefix = "zarf-"
)

// GetCryptoHashFromFile returns the computed SHA256 Sum of a given file
func GetCryptoHashFromFile(path string, hashName crypto.Hash) (string, error) {
	var data io.ReadCloser
	var err error

	if IsURL(path) {
		// Handle download from URL
		message.Warn("This is a remote source. If a published checksum is available you should use that rather than calculating it directly from the remote link.")
		data = Fetch(path)
	} else {
		// Handle local file
		data, err = os.Open(path)
		if err != nil {
			return "", err
		}
	}

	defer data.Close()
	return helpers.GetCryptoHash(data, hashName)
}

// TextTemplate represents a value to be templated into a text file.
type TextTemplate struct {
	Sensitive  bool
	AutoIndent bool
	Type       types.VariableType
	Value      string
}

// MakeTempDir creates a temp directory with the given prefix.
func MakeTempDir(tmpDir string) (string, error) {
	// Create the base tmp directory if it is specified.
	if tmpDir != "" {
		if err := CreateDirectory(tmpDir, 0700); err != nil {
			return "", err
		}
	}

	tmp, err := os.MkdirTemp(tmpDir, tmpPathPrefix)
	message.Debugf("Using temp path: '%s'", tmp)
	return tmp, err
}

// VerifyBinary returns true if binary is available.
func VerifyBinary(binary string) bool {
	_, err := exec.LookPath(binary)
	return err == nil
}

// CreateDirectory creates a directory for the given path and file mode.
func CreateDirectory(path string, mode os.FileMode) error {
	if InvalidPath(path) {
		return os.MkdirAll(path, mode)
	}
	return nil
}

// CreateFile creates an empty file at the given path.
func CreateFile(filepath string) error {
	if InvalidPath(filepath) {
		f, err := os.Create(filepath)
		f.Close()
		return err
	}

	return nil

}

// InvalidPath checks if the given path is valid (if it is a permissions error it is there we just don't have access)
func InvalidPath(path string) bool {
	_, err := os.Stat(path)
	return !os.IsPermission(err) && err != nil
}

// ListDirectories returns a list of directories in the given directory.
func ListDirectories(directory string) ([]string, error) {
	var directories []string
	paths, err := os.ReadDir(directory)
	if err != nil {
		return directories, fmt.Errorf("unable to load the directory %s: %w", directory, err)
	}

	for _, entry := range paths {
		if entry.IsDir() {
			directories = append(directories, filepath.Join(directory, entry.Name()))
		}
	}

	return directories, nil
}

// WriteFile writes the given data to the given path.
func WriteFile(path string, data []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("unable to create the file at %s to write the contents: %w", path, err)
	}

	_, err = f.Write(data)
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("unable to write the file at %s contents:%w", path, err)
	}

	err = f.Close()
	if err != nil {
		return fmt.Errorf("error saving file %s: %w", path, err)
	}

	return nil
}

// ReplaceTextTemplate loads a file from a given path, replaces text in it and writes it back in place.
func ReplaceTextTemplate(path string, mappings map[string]*TextTemplate, deprecations map[string]string, templateRegex string) error {
	textFile, err := os.Open(path)
	if err != nil {
		return err
	}

	// This regex takes a line and parses the text before and after a discovered template: https://regex101.com/r/ilUxAz/1
	regexTemplateLine := regexp.MustCompile(fmt.Sprintf("(?P<preTemplate>.*?)(?P<template>%s)(?P<postTemplate>.*)", templateRegex))

	fileScanner := bufio.NewScanner(textFile)

	// Set the buffer to 1 MiB to handle long lines (i.e. base64 text in a secret)
	// 1 MiB is around the documented maximum size for secrets and configmaps
	const maxCapacity = 1024 * 1024
	buf := make([]byte, maxCapacity)
	fileScanner.Buffer(buf, maxCapacity)

	// Set the scanner to split on new lines
	fileScanner.Split(bufio.ScanLines)

	text := ""

	for fileScanner.Scan() {
		line := fileScanner.Text()

		for {
			matches := regexTemplateLine.FindStringSubmatch(line)

			// No template left on this line so move on
			if len(matches) == 0 {
				text += fmt.Sprintln(line)
				break
			}

			preTemplate := matches[regexTemplateLine.SubexpIndex("preTemplate")]
			templateKey := matches[regexTemplateLine.SubexpIndex("template")]

			_, present := deprecations[templateKey]
			if present {
				message.Warnf("This Zarf Package uses a deprecated variable: '%s' changed to '%s'.  Please notify your package creator for an update.", templateKey, deprecations[templateKey])
			}

			template := mappings[templateKey]

			// Check if the template is nil (present), use the original templateKey if not (so that it is not replaced).
			value := templateKey
			if template != nil {
				value = template.Value

				// Check if the value is a file type and load the value contents from the file
				if template.Type == types.FileVariableType {
					if isText, err := IsTextFile(value); err != nil || !isText {
						message.Warnf("Refusing to load a non-text file for templating %s", templateKey)
						line = matches[regexTemplateLine.SubexpIndex("postTemplate")]
						continue
					}

					contents, err := os.ReadFile(value)
					if err != nil {
						message.Warnf("Unable to read file for templating - skipping: %s", err.Error())
						line = matches[regexTemplateLine.SubexpIndex("postTemplate")]
						continue
					}

					value = string(contents)
				}

				// Check if the value is autoIndented and add the correct spacing
				if template.AutoIndent {
					indent := fmt.Sprintf("\n%s", strings.Repeat(" ", len(preTemplate)))
					value = strings.ReplaceAll(value, "\n", indent)
				}
			}

			// Add the processed text and continue processing the line
			text += fmt.Sprintf("%s%s", preTemplate, value)
			line = matches[regexTemplateLine.SubexpIndex("postTemplate")]
		}
	}

	textFile.Close()

	return os.WriteFile(path, []byte(text), 0600)

}

// RecursiveFileList walks a path with an optional regex pattern and returns a slice of file paths.
// If skipHidden is true, hidden directories will be skipped.
func RecursiveFileList(dir string, pattern *regexp.Regexp, skipHidden bool) (files []string, err error) {
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {

		// Return errors
		if err != nil {
			return err
		}

		if !d.IsDir() {
			if pattern != nil {
				if len(pattern.FindStringIndex(path)) > 0 {
					files = append(files, path)
				}
			} else {
				files = append(files, path)
			}
			// Skip hidden directories
		} else if skipHidden && IsHidden(d.Name()) {
			return filepath.SkipDir
		}

		return nil
	})
	return files, err
}

// CreateFilePath creates the parent directory for the given file path.
func CreateFilePath(destination string) error {
	parentDest := filepath.Dir(destination)
	return CreateDirectory(parentDest, 0700)
}

// CreatePathAndCopy creates the parent directory for the given file path and copies the source file to the destination.
func CreatePathAndCopy(source string, destination string) error {
	if err := CreateFilePath(destination); err != nil {
		return err
	}

	// Copy all the source data into the destination location
	if err := copy.Copy(source, destination); err != nil {
		return err
	}

	// If the path doesn't exist yet then this is an empty file and we should create it
	return CreateFile(destination)
}

// GetFinalExecutablePath returns the absolute path to the Zarf executable, following any symlinks along the way.
func GetFinalExecutablePath() (string, error) {
	message.Debug("utils.GetExecutablePath()")

	binaryPath, err := os.Executable()
	if err != nil {
		return "", err
	}

	// In case the binary is symlinked somewhere else, get the final destination!!
	linkedPath, err := filepath.EvalSymlinks(binaryPath)
	return linkedPath, err
}

// SplitFile splits a file into multiple parts by the given size.
func SplitFile(path string, chunkSizeBytes int) (chunks [][]byte, sha256sum string, err error) {
	var file []byte

	// Open the created archive for io.Copy
	if file, err = os.ReadFile(path); err != nil {
		return chunks, sha256sum, err
	}

	//Calculate the sha256sum of the file before we split it up
	sha256sum = fmt.Sprintf("%x", sha256.Sum256(file))

	// Loop over the tarball breaking it into chunks based on the payloadChunkSize
	for {
		if len(file) == 0 {
			break
		}

		// don't bust slice length
		if len(file) < chunkSizeBytes {
			chunkSizeBytes = len(file)
		}

		chunks = append(chunks, file[0:chunkSizeBytes])
		file = file[chunkSizeBytes:]
	}

	return chunks, sha256sum, nil
}

// IsTextFile returns true if the given file is a text file.
func IsTextFile(path string) (bool, error) {
	// Open the file
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close() // Make sure to close the file when we're done

	// Read the first 512 bytes of the file
	data := make([]byte, 512)
	n, err := f.Read(data)
	if err != nil && err != io.EOF {
		return false, err
	}

	// Use http.DetectContentType to determine the MIME type of the file
	mimeType := http.DetectContentType(data[:n])

	// Check if the MIME type indicates that the file is text
	hasText := strings.HasPrefix(mimeType, "text/")
	hasJSON := strings.Contains(mimeType, "json")
	hasXML := strings.Contains(mimeType, "xml")

	return hasText || hasJSON || hasXML, nil
}

// IsTrashBin checks if the given directory path corresponds to an operating system's trash bin.
func IsTrashBin(dirPath string) bool {
	dirPath = filepath.Clean(dirPath)

	// Check if the directory path matches a Linux trash bin
	if strings.HasSuffix(dirPath, "/Trash") || strings.HasSuffix(dirPath, "/.Trash-1000") {
		return true
	}

	// Check if the directory path matches a macOS trash bin
	if strings.HasSuffix(dirPath, "./Trash") || strings.HasSuffix(dirPath, "/.Trashes") {
		return true
	}

	// Check if the directory path matches a Windows trash bin
	if strings.HasSuffix(dirPath, "\\$RECYCLE.BIN") {
		return true
	}

	return false
}

// IsHidden returns true if the given file name starts with a dot.
func IsHidden(name string) bool {
	return name[0] == dotCharacter
}

// GetDirSize walks through all files and directories in the provided path and returns the total size in bytes.
func GetDirSize(path string) (int64, error) {
	dirSize := int64(0)

	// Walk all files in the path
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			dirSize += info.Size()
		}
		return nil
	})

	return dirSize, err
}

// IsDir returns true if the given path is a directory.
func IsDir(path string) bool {
	info, err := os.Stat(filepath.Clean(path))
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// GetSHA256OfFile returns the SHA256 hash of the provided file.
func GetSHA256OfFile(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	return helpers.GetCryptoHash(f, crypto.SHA256)
}
