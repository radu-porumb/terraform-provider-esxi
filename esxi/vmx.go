package esxi

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

// ParseVmxFile parses VMX file to map
func ParseVmxFile(contents string) map[string]string {
	results := make(map[string]string)

	lineRe := regexp.MustCompile(`^(.+?)\s*=\s*"(.*?)"\s*$`)

	for _, line := range strings.Split(contents, "\n") {
		matches := lineRe.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		results[matches[1]] = matches[2]
	}

	return results
}

// BuildVmxString builds a valid VMX file from a data map
func BuildVmxString(contents map[string]string) string {
	var buf bytes.Buffer

	i := 0
	keys := make([]string, len(contents))
	for k := range contents {
		keys[i] = k
		i++
	}

	sort.Strings(keys)
	for _, k := range keys {
		buf.WriteString(fmt.Sprintf("%s = \"%s\"\n", k, contents[k]))
	}

	return buf.String()
}

// SaveVmxDataToDisk saves a map of VMX contents to disk
func SaveVmxDataToDisk(path string, data map[string]string) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()

	var buf bytes.Buffer
	buf.WriteString(BuildVmxString(data))
	if _, err = io.Copy(f, &buf); err != nil {
		return
	}

	return
}

// SaveVmxStringToDisk saves VMX contents string to disk
func SaveVmxStringToDisk(fileName string, data string) (err error) {

	file, err := os.Create(fileName)
	if err != nil {
		return
	}
	defer file.Close()

	_, err = io.WriteString(file, data)

	if err != nil {
		return
	}
	return file.Sync()
}

// DeleteVmx deletes VMX file from local disk
func DeleteVmx(fileName string) (err error) {
	err = os.Remove(fileName)

	return
}

// GetVmxFileFromPath gets the VMX file name from a path string
func GetVmxFileFromPath(path string) string {
	if !strings.Contains(path, "/") {
		return path
	}

	pathFragments := strings.Split(path, "/")

	return pathFragments[len(pathFragments)-1]
}
