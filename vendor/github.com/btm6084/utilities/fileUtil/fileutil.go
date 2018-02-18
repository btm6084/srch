package fileUtil

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// IsBinaryFile attemtpts to determine whether a file is a binary file.
func IsBinaryFile(fileName string) bool {
	if IsDir(fileName) {
		return false
	}

	file, err := os.Open(fileName)
	if err != nil {
		return true
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if !utf8.Valid([]byte(line)) {
			return true
		}
	}

	if err := scanner.Err(); err != nil {
		return false
	}

	return false
}

// IsDir will return true only if the given path exists and is a directory.
func IsDir(fileName string) bool {
	f, err := os.Stat(fileName)
	if err != nil {
		log.Println(err)
		return false
	}

	if !f.Mode().IsDir() {
		return false
	}

	return true
}

// IsSymlink returns if a given file is a symlink, False otherwise.
func IsSymlink(fileName string) bool {
	f, err := os.Lstat(fileName)
	if err != nil {
		log.Println(err)
		return false
	}

	return (f.Mode() & os.ModeSymlink) != 0
}

// DirToArray will flatten a directory into a simple list of files that can be iterated over.
func DirToArray(dir string, followSyms bool, fileFilter func(string, string) bool, dirFilter func(string, string) bool) []string {
	var results []string

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return results
	}

	for _, f := range files {
		file := fmt.Sprintf("%s/%s", strings.TrimRight(dir, "/"), strings.TrimLeft(f.Name(), "/"))

		if !followSyms && IsSymlink(file) {
			continue
		}

		if IsDir(file) {
			// Don't mess with .git files.
			if f.Name() == ".git" {
				continue
			}

			// Run the directory filter
			if !dirFilter(dir, f.Name()) {
				continue
			}

			subFiles := DirToArray(file, followSyms, fileFilter, dirFilter)
			results = append(results, subFiles...)
			continue
		}

		// filter should return true if the file should be kept, false if it should be discarded.
		if fileFilter(dir, f.Name()) {
			results = append(results, file)
		}
	}

	return results
}

// DefaultDirectoryFilter returns true if the file should be kept, false if it should be discarded.
func DefaultDirectoryFilter(path, dirName string) bool {
	if dirName == ".git" {
		return false
	}

	return true
}

// DefaultFileFilter returns true if the file should be kept, false if it should be discarded.
func DefaultFileFilter(path, fileName string) bool {
	file := fmt.Sprintf("%s/%s", path, fileName)

	if IsSymlink(file) || IsBinaryFile(file) {
		return false
	}

	return true
}

// FilterOutBinaries removes any binary files from the given file list.
func FilterOutBinaries(files []string) []string {
	var fileList []string

	for _, f := range files {
		if !IsBinaryFile(f) {
			fileList = append(fileList, f)
		}
	}

	return fileList
}

// FilterOutSymlinks removes any symlink files from the given file list.
func FilterOutSymlinks(files []string) []string {
	var fileList []string

	for _, f := range files {
		if !IsSymlink(f) {
			fileList = append(fileList, f)
		}
	}

	return fileList
}

// FilterExtWhitelist removes any files where the extension is not found in the extension list.
func FilterExtWhitelist(ext, files []string) []string {
	var fileList []string
	for _, f := range files {
		for _, e := range ext {
			e1 := strings.TrimLeft(e, ".")
			e2 := strings.TrimLeft(filepath.Ext(f), ".")

			if e1 == e2 {
				fileList = append(fileList, f)
			}
		}
	}

	return fileList
}

// FilterExtBlacklist removes any files where the extension is found in the extension list.
func FilterExtBlacklist(ext, files []string) []string {
	var fileList []string
	for _, f := range files {
		found := false
		for _, e := range ext {
			e1 := strings.TrimLeft(e, ".")
			e2 := strings.TrimLeft(filepath.Ext(f), ".")

			if e1 == e2 {
				found = true
			}
		}

		if !found {
			fileList = append(fileList, f)
		}
	}

	return fileList
}
