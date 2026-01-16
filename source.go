package sloghandler

import (
	"path/filepath"
)

type sourceCacheKey struct {
	path  string
	depth int
}

func (h *logHandler) getFilePath(path string) []byte {
	cacheKey := sourceCacheKey{path: path, depth: h.opts.SourceDepth}
	if cached, ok := h.sourceCache.Load(cacheKey); ok {
		return cached.([]byte)
	}

	depth := h.opts.SourceDepth
	if depth < 0 {
		depth = 0 // Default to 0 if negative
	}

	if depth == 0 {
		// Show only filename
		result := []byte(filepath.Base(path))
		h.sourceCache.Store(cacheKey, result)
		return result
	}

	// Build path with specified depth
	parts := []string{filepath.Base(path)} // Start with filename
	currentPath := filepath.Dir(path)

	for i := 0; i < depth && currentPath != "." && currentPath != string(filepath.Separator) && currentPath != "" && filepath.Dir(currentPath) != currentPath; i++ {
		parts = append([]string{filepath.Base(currentPath)}, parts...)
		currentPath = filepath.Dir(currentPath)
	}

	result := []byte(filepath.Join(parts...))
	h.sourceCache.Store(cacheKey, result)
	return result
}
