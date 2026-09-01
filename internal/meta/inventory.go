package meta

import (
	"os"
	"path/filepath"
	"strings"
)

func CollectInventory(root string) (Inventory, []string, error) {
	var inventory Inventory
	generated := []string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == filepath.Join(root, ".git") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			if path != root {
				inventory.DescendantDirs++
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if relative == "README.md" {
			inventory.RootReadmeExcluded = 1
			return nil
		}
		inventory.RegularFiles++
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.HasSuffix(relative, ".go") {
			inventory.GoFiles++
			inventory.GoPhysicalLines += physicalLines(raw)
		}
		if strings.HasSuffix(relative, ".gooo") {
			inventory.GoooFiles++
			inventory.GoooPhysicalLines += physicalLines(raw)
		}
		if isGeneratedPath(relative) {
			inventory.GeneratedFiles++
			inventory.GeneratedBytes += int64(len(raw))
			generated = append(generated, filepath.ToSlash(relative))
		}
		return nil
	})
	return inventory, generated, err
}

func isGeneratedPath(relative string) bool {
	return strings.HasPrefix(filepath.ToSlash(relative), "internal/generated/") || strings.HasSuffix(relative, ".generated.go")
}

func physicalLines(raw []byte) int {
	if len(raw) == 0 {
		return 0
	}
	count := 1
	for _, character := range raw {
		if character == '\n' {
			count++
		}
	}
	if raw[len(raw)-1] == '\n' {
		count--
	}
	return count
}
