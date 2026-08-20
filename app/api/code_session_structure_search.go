package api

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/aihop/gopanel/app/e"
	"github.com/gofiber/fiber/v3"
)

const (
	maxAIStructureSearchHits     = 80
	maxAIStructureSearchFiles    = 8000
	maxAIStructureSearchFileSize = 256 * 1024
	minAIStructureSearchRunes    = 2
)

var generatedAIStructureDirNames = map[string]struct{}{
	".cache": {}, ".gradle": {}, ".next": {}, ".nuxt": {}, ".output": {},
	".parcel-cache": {}, ".pnpm-store": {}, ".svelte-kit": {}, ".turbo": {}, ".venv": {},
	".vite": {}, "__pycache__": {}, "build": {}, "coverage": {}, "dist": {},
	"node_modules": {}, "out": {}, "Pods": {}, "target": {}, "tmp": {}, "vendor": {},
}

func skipAIStructureGitMetadata(name string) bool {
	return name == ".git"
}

var generatedAIStructureFileNames = map[string]struct{}{
	".gopanel-project.json": {},
	"composer.lock":         {},
	"package-lock.json":     {},
	"pnpm-lock.yaml":        {},
	"yarn.lock":             {},
}

var skippedAIStructureContentExt = map[string]struct{}{
	".7z": {}, ".bin": {}, ".bmp": {}, ".class": {}, ".dll": {}, ".eot": {}, ".exe": {},
	".gif": {}, ".gz": {}, ".ico": {}, ".jar": {}, ".jpeg": {}, ".jpg": {}, ".map": {},
	".mp3": {}, ".mp4": {}, ".o": {}, ".otf": {}, ".pdf": {}, ".png": {}, ".so": {},
	".tar": {}, ".ttf": {}, ".wasm": {}, ".wav": {}, ".webm": {}, ".webp": {},
	".woff": {}, ".woff2": {}, ".zip": {},
}

type codeStructureSearchHit struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	IsDir     bool   `json:"isDir"`
	Extension string `json:"extension"`
	Kind      string `json:"kind"`
	Line      int    `json:"line,omitempty"`
	Preview   string `json:"preview,omitempty"`
}

type codeStructureSearchResult struct {
	Query     string                   `json:"query"`
	Hits      []codeStructureSearchHit `json:"hits"`
	Truncated bool                     `json:"truncated"`
}

func skipGeneratedAIStructureDir(name string) bool {
	if skipAIStructureGitMetadata(name) {
		return true
	}
	if name == ".gopanel-project.json" {
		return true
	}
	_, generated := generatedAIStructureDirNames[name]
	return generated
}

func skipGeneratedAIStructureFile(name string) bool {
	lower := strings.ToLower(name)
	if _, skip := generatedAIStructureFileNames[lower]; skip {
		return true
	}
	if strings.HasSuffix(lower, ".min.js") || strings.HasSuffix(lower, ".min.css") || strings.HasSuffix(lower, ".map") {
		return true
	}
	_, skip := skippedAIStructureContentExt[strings.ToLower(filepath.Ext(name))]
	return skip
}

func structureSearchPreview(content, query string) (int, string) {
	at := strings.Index(strings.ToLower(content), strings.ToLower(query))
	if at < 0 {
		return 0, ""
	}
	lineStart := strings.LastIndex(content[:at], "\n") + 1
	lineEnd := strings.Index(content[at:], "\n")
	if lineEnd < 0 {
		lineEnd = len(content)
	} else {
		lineEnd += at
	}
	preview := strings.TrimSpace(content[lineStart:lineEnd])
	runes := []rune(preview)
	if len(runes) > 80 {
		preview = string(runes[:80]) + "…"
	}
	return 1 + strings.Count(content[:at], "\n"), preview
}

func searchAISessionStructure(workDir, query, relativePath string, sourceDirs []string) (codeStructureSearchResult, error) {
	query = strings.TrimSpace(query)
	result := codeStructureSearchResult{Query: query, Hits: []codeStructureSearchHit{}}
	if utf8.RuneCountInString(query) < minAIStructureSearchRunes {
		return result, nil
	}
	root, startRel, roots, err := resolveAIStructurePath(workDir, relativePath, sourceDirs)
	if err != nil {
		return result, err
	}
	if startRel == "." {
		startRel = ""
	}
	scanned := 0
	var walk func(dir, rel string) error
	walk = func(dir, rel string) error {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}
		for _, entry := range entries {
			if len(result.Hits) >= maxAIStructureSearchHits || scanned >= maxAIStructureSearchFiles {
				result.Truncated = true
				return fs.SkipAll
			}
			name := entry.Name()
			childRel := name
			if rel != "" {
				childRel = path.Join(rel, name)
			}
			if skipAIStructureGitMetadata(name) {
				continue
			}
			if skipGeneratedAIStructureDir(name) {
				continue
			}
			child := filepath.Join(dir, name)
			resolved, resolveErr := filepath.EvalSymlinks(child)
			if resolveErr != nil || !isPathWithinAnyRoot(filepath.Clean(resolved), roots) {
				continue
			}
			info, statErr := os.Stat(resolved)
			if statErr != nil {
				continue
			}
			if info.IsDir() {
				if strings.Contains(strings.ToLower(name), strings.ToLower(query)) {
					result.Hits = append(result.Hits, codeStructureSearchHit{
						Name: name, Path: childRel, IsDir: true, Kind: "name",
					})
				}
				if err := walk(resolved, childRel); err != nil {
					return err
				}
				continue
			}
			scanned++
			hit := codeStructureSearchHit{
				Name:      name,
				Path:      childRel,
				Extension: strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), "."),
			}
			nameMatched := strings.Contains(strings.ToLower(name), strings.ToLower(query))
			if skipGeneratedAIStructureFile(name) || info.Size() > maxAIStructureSearchFileSize {
				if nameMatched {
					hit.Kind = "name"
					result.Hits = append(result.Hits, hit)
				}
				continue
			}
			content, readErr := os.ReadFile(resolved)
			if readErr == nil && utf8.Valid(content) && strings.IndexByte(string(content), 0) < 0 {
				line, preview := structureSearchPreview(string(content), query)
				if line > 0 {
					hit.Line = line
					hit.Preview = preview
					if !nameMatched {
						hit.Kind = "content"
					}
				}
			}
			if nameMatched {
				hit.Kind = "name"
				result.Hits = append(result.Hits, hit)
				continue
			}
			if hit.Kind == "content" {
				result.Hits = append(result.Hits, hit)
			}
		}
		return nil
	}
	if err := walk(root, startRel); err != nil && err != fs.SkipAll {
		return result, err
	}
	sort.SliceStable(result.Hits, func(left, right int) bool {
		if result.Hits[left].Kind != result.Hits[right].Kind {
			return result.Hits[left].Kind == "name"
		}
		return result.Hits[left].Path < result.Hits[right].Path
	})
	return result, nil
}

func GetAISessionStructureSearch(c fiber.Ctx) error {
	_, workDir, sourceDirs, err := getAISessionFileContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	result, err := searchAISessionStructure(workDir, c.Query("q"), c.Query("path"), sourceDirs)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(result))
}
