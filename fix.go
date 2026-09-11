package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"

	"honnef.co/go/tools/sarif"
)

type suggestedFileEdit struct {
	Path string
	Edit fileEdit
}

type fixSource struct {
	Data       []byte
	LineStarts []int
	Mode       os.FileMode
	Generated  bool
}

func applySuggestedFixes(output []byte) (int, error) {
	var log sarif.Log

	err := json.Unmarshal(output, &log)
	if err != nil {
		return 0, fmt.Errorf("decode SARIF: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return 0, fmt.Errorf("get working directory: %w", err)
	}

	sources := make(map[string]*fixSource)
	selected := make(map[string][]fileEdit)
	seen := make(map[string]struct{})
	fixedCount := 0

	for _, run := range log.Runs {
		for _, result := range run.Results {
			if len(result.Fixes) != 1 {
				continue
			}

			candidate, valid, prepareErr := prepareSuggestedFix(cwd, result.Fixes[0], sources)
			if prepareErr != nil {
				return 0, prepareErr
			}

			if !valid {
				continue
			}

			key := suggestedFixKey(candidate)

			_, duplicate := seen[key]
			if duplicate {
				continue
			}

			seen[key] = struct{}{}

			if suggestedFixConflicts(candidate, selected) {
				continue
			}

			added := false

			for _, edit := range candidate {
				if containsFileEdit(selected[edit.Path], edit.Edit) {
					continue
				}

				selected[edit.Path] = append(selected[edit.Path], edit.Edit)
				added = true
			}

			if added {
				fixedCount++
			}
		}
	}

	paths := slices.Sorted(maps.Keys(selected))
	pending := make([]fixedFile, 0, len(paths))

	for _, path := range paths {
		updated, applyErr := applyFileEdits(sources[path].Data, selected[path])
		if applyErr != nil {
			return 0, fmt.Errorf("apply edits to %s: %w", path, applyErr)
		}

		pending = append(pending, fixedFile{
			Path: path,
			Data: updated,
			Mode: sources[path].Mode,
		})
	}

	for _, file := range pending {
		err = os.WriteFile(file.Path, file.Data, file.Mode)
		if err != nil {
			return 0, fmt.Errorf("write %s: %w", file.Path, err)
		}
	}

	return fixedCount, nil
}

func prepareSuggestedFix(cwd string, fix sarif.Fix, sources map[string]*fixSource) ([]suggestedFileEdit, bool, error) {
	replacementCount := 0

	for _, change := range fix.ArtifactChanges {
		replacementCount += len(change.Replacements)
	}

	if replacementCount == 0 {
		return nil, false, nil
	}

	edits := make([]suggestedFileEdit, 0, replacementCount)

	for _, change := range fix.ArtifactChanges {
		path, err := resolveArtifactPath(cwd, change.ArtifactLocation.URI)
		if err != nil {
			return nil, false, fmt.Errorf("resolve fix path %q: %w", change.ArtifactLocation.URI, err)
		}

		source := sources[path]
		if source == nil {
			source, err = loadFixSource(path)
			if err != nil {
				return nil, false, err
			}

			sources[path] = source
		}

		if source.Generated {
			return nil, false, nil
		}

		for _, replacement := range change.Replacements {
			edit, ok := replacementEdit(source, replacement)
			if !ok {
				return nil, false, nil
			}

			candidate := suggestedFileEdit{Path: path, Edit: edit}

			for _, existing := range edits {
				if existing.Path == path && fileEditsConflict(existing.Edit, edit) {
					return nil, false, nil
				}
			}

			edits = append(edits, candidate)
		}
	}

	slices.SortFunc(edits, func(first, second suggestedFileEdit) int {
		return cmp.Or(
			cmp.Compare(first.Path, second.Path),
			cmp.Compare(first.Edit.Start, second.Edit.Start),
			cmp.Compare(first.Edit.End, second.Edit.End),
			cmp.Compare(first.Edit.NewText, second.Edit.NewText),
		)
	})

	return edits, true, nil
}

func resolveArtifactPath(cwd, uri string) (string, error) {
	if uri == "" {
		return "", fmt.Errorf("artifact URI is empty")
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return "", err
	}

	if parsed.Scheme != "" && parsed.Scheme != "file" {
		return "", fmt.Errorf("unsupported URI scheme %q", parsed.Scheme)
	}

	path := parsed.Path

	if parsed.Scheme == "file" && parsed.Host != "" {
		path = "//" + parsed.Host + path
	}

	if runtime.GOOS == "windows" && len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}

	path = filepath.FromSlash(path)

	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}

	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return filepath.Clean(path), nil
}

func loadFixSource(path string) (*fixSource, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	lineStarts := make([]int, 1, len(data)/32+1)

	for index, value := range data {
		if value == '\n' {
			lineStarts = append(lineStarts, index+1)
		}
	}

	return &fixSource{
		Data:       data,
		LineStarts: lineStarts,
		Mode:       info.Mode(),
		Generated:  isGeneratedFile(path),
	}, nil
}

func replacementEdit(source *fixSource, replacement sarif.Replacement) (fileEdit, bool) {
	region := replacement.DeletedRegion

	start, ok := sourceOffset(source, region.StartLine, region.StartColumn)
	if !ok {
		return fileEdit{}, false
	}

	endLine := region.EndLine
	endColumn := region.EndColumn

	if endLine == 0 {
		endLine = region.StartLine
	}

	if endColumn == 0 {
		endColumn = region.StartColumn
	}

	end, ok := sourceOffset(source, endLine, endColumn)
	if !ok || end < start {
		return fileEdit{}, false
	}

	return fileEdit{Start: start, End: end, NewText: replacement.InsertedContent.Text}, true
}

func sourceOffset(source *fixSource, line, column int) (int, bool) {
	if line < 1 || line > len(source.LineStarts) || column < 1 {
		return 0, false
	}

	offset := source.LineStarts[line-1] + column - 1
	if offset < 0 || offset > len(source.Data) {
		return 0, false
	}

	if line < len(source.LineStarts) && offset >= source.LineStarts[line] {
		return 0, false
	}

	return offset, true
}

func suggestedFixConflicts(candidate []suggestedFileEdit, selected map[string][]fileEdit) bool {
	for _, edit := range candidate {
		for _, existing := range selected[edit.Path] {
			if fileEditsConflict(existing, edit.Edit) {
				return true
			}
		}
	}

	return false
}

func fileEditsConflict(first, second fileEdit) bool {
	if first == second {
		return false
	}

	if first.Start == second.Start && (first.Start == first.End || second.Start == second.End) {
		return true
	}

	return first.Start < second.End && second.Start < first.End
}

func containsFileEdit(edits []fileEdit, target fileEdit) bool {
	return slices.Contains(edits, target)
}

func suggestedFixKey(edits []suggestedFileEdit) string {
	key := make([]byte, 0, len(edits)*32)

	for _, edit := range edits {
		key = appendKeyString(key, edit.Path)
		key = strconv.AppendInt(key, int64(edit.Edit.Start), 10)
		key = append(key, ':')
		key = strconv.AppendInt(key, int64(edit.Edit.End), 10)
		key = append(key, ':')
		key = appendKeyString(key, edit.Edit.NewText)
	}

	return string(key)
}

func appendKeyString(dst []byte, value string) []byte {
	dst = strconv.AppendInt(dst, int64(len(value)), 10)
	dst = append(dst, ':')
	dst = append(dst, value...)

	return append(dst, ';')
}

func applyFileEdits(contents []byte, edits []fileEdit) ([]byte, error) {
	slices.SortFunc(edits, func(first, second fileEdit) int {
		return cmp.Or(
			cmp.Compare(first.Start, second.Start),
			cmp.Compare(first.End, second.End),
		)
	})

	capacity := len(contents)

	for _, edit := range edits {
		if edit.Start < 0 || edit.End < edit.Start || edit.End > len(contents) {
			return nil, fmt.Errorf("invalid edit range %d:%d", edit.Start, edit.End)
		}

		capacity += len(edit.NewText) - (edit.End - edit.Start)
	}

	updated := make([]byte, 0, capacity)
	cursor := 0

	for _, edit := range edits {
		if edit.Start < cursor {
			return nil, fmt.Errorf("overlapping edit at %d:%d", edit.Start, edit.End)
		}

		updated = append(updated, contents[cursor:edit.Start]...)
		updated = append(updated, edit.NewText...)
		cursor = edit.End
	}

	updated = append(updated, contents[cursor:]...)

	return updated, nil
}
