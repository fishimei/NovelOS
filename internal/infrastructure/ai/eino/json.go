package eino

import (
	"encoding/json"
	"fmt"
	"strings"
)

type modelJSONMeta struct {
	RawLength       int
	ExtractedLength int
	FencedRemoved   bool
	FoundObject     bool
	RawPrefix       string
}

type modelJSONError struct {
	Meta modelJSONMeta
	Err  error
}

func (e modelJSONError) Error() string {
	return e.Err.Error()
}

func (e modelJSONError) Unwrap() error {
	return e.Err
}

func decodeModelJSON[T any](content string, target *T) error {
	_, err := decodeModelJSONWithMeta(content, target)
	return err
}

func decodeModelJSONWithMeta[T any](content string, target *T) (modelJSONMeta, error) {
	text, meta := extractModelJSON(content)
	if err := json.Unmarshal([]byte(text), target); err != nil {
		return meta, modelJSONError{Meta: meta, Err: err}
	}
	return meta, nil
}

func extractModelJSON(content string) (string, modelJSONMeta) {
	text := strings.TrimSpace(content)
	meta := modelJSONMeta{RawLength: len(content), RawPrefix: truncateForAudit(text, 240)}
	if strings.HasPrefix(text, "```json") {
		text = strings.TrimPrefix(text, "```json")
		meta.FencedRemoved = true
	} else if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		meta.FencedRemoved = true
	}
	if strings.HasSuffix(text, "```") {
		text = strings.TrimSuffix(text, "```")
		meta.FencedRemoved = true
	}
	text = strings.TrimSpace(text)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end >= start {
		text = text[start : end+1]
		meta.FoundObject = true
	}
	meta.ExtractedLength = len(text)
	return text, meta
}

func modelJSONErrorSummary(err error) map[string]any {
	if parseErr, ok := err.(modelJSONError); ok {
		return map[string]any{
			"parse_error":      parseErr.Err.Error(),
			"raw_length":       parseErr.Meta.RawLength,
			"extracted_length": parseErr.Meta.ExtractedLength,
			"fenced_removed":   parseErr.Meta.FencedRemoved,
			"found_object":     parseErr.Meta.FoundObject,
			"raw_prefix":       parseErr.Meta.RawPrefix,
		}
	}
	return map[string]any{"parse_error": fmt.Sprint(err)}
}

func truncateForAudit(text string, limit int) string {
	if limit <= 0 || len(text) <= limit {
		return text
	}
	return text[:limit]
}
