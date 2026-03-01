package prompts

// GenerateResultsSchema returns the JSON schema for content generation results.
// The API requires a top-level object, so results are wrapped in {"results": [...]}.
func GenerateResultsSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"results": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"content":          map[string]any{"type": "string"},
						"viral_mechanic":   map[string]any{"type": "string"},
						"confidence_score": map[string]any{"type": "integer"},
					},
					"required":             []string{"content", "viral_mechanic", "confidence_score"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"results"},
		"additionalProperties": false,
	}
}

// RepoGenerateResultsSchema returns the JSON schema for repo-to-post generation
// with an additional code_snippet field per result for AI-driven code image selection.
func RepoGenerateResultsSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"results": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"content":          map[string]any{"type": "string"},
						"viral_mechanic":   map[string]any{"type": "string"},
						"confidence_score": map[string]any{"type": "integer"},
						"code_snippet": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"filename":          map[string]any{"type": "string"},
								"start_line":        map[string]any{"type": "integer"},
								"end_line":          map[string]any{"type": "integer"},
								"image_description": map[string]any{"type": "string"},
							},
							"required":             []string{"filename", "start_line", "end_line", "image_description"},
							"additionalProperties": false,
						},
					},
					"required":             []string{"content", "viral_mechanic", "confidence_score", "code_snippet"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"results"},
		"additionalProperties": false,
	}
}

// ClassifySingleSchema returns the JSON schema for classifying a single post.
func ClassifySingleSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"decision":   map[string]any{"type": "string"},
			"reasoning":  map[string]any{"type": "string"},
			"confidence": map[string]any{"type": "integer"},
		},
		"required":             []string{"decision", "reasoning", "confidence"},
		"additionalProperties": false,
	}
}

// ClassifyBatchSchema returns the JSON schema for classifying multiple posts.
// Results are wrapped in {"results": [...]}.
func ClassifyBatchSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"results": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"decision":   map[string]any{"type": "string"},
						"reasoning":  map[string]any{"type": "string"},
						"confidence": map[string]any{"type": "integer"},
					},
					"required":             []string{"decision", "reasoning", "confidence"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"results"},
		"additionalProperties": false,
	}
}

// ActionSelectBatchSchema returns the JSON schema for batch action selection results.
// Results are wrapped in {"results": [...]}.
func ActionSelectBatchSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"results": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"action":     map[string]any{"type": "string"},
						"reasoning":  map[string]any{"type": "string"},
						"confidence": map[string]any{"type": "integer"},
					},
					"required":             []string{"action", "reasoning", "confidence"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"results"},
		"additionalProperties": false,
	}
}

// PersonaProfileSchema returns the JSON schema for persona profile analysis.
func PersonaProfileSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"writing_tone":        map[string]any{"type": "string"},
			"typical_length":      map[string]any{"type": "string"},
			"common_themes":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"vocabulary_level":    map[string]any{"type": "string"},
			"engagement_patterns": map[string]any{"type": "string"},
			"structural_patterns": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"emoji_usage":         map[string]any{"type": "string"},
			"hashtag_usage":       map[string]any{"type": "string"},
			"call_to_action_style": map[string]any{"type": "string"},
			"unique_quirks":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"voice_summary":       map[string]any{"type": "string"},
		},
		"required": []string{
			"writing_tone", "typical_length", "common_themes", "vocabulary_level",
			"engagement_patterns", "structural_patterns", "emoji_usage", "hashtag_usage",
			"call_to_action_style", "unique_quirks", "voice_summary",
		},
		"additionalProperties": false,
	}
}

// ImageDecisionSchema returns the JSON schema for image decision output.
func ImageDecisionSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"suggest_image": map[string]any{"type": "boolean"},
			"reasoning":     map[string]any{"type": "string"},
		},
		"required":             []string{"suggest_image", "reasoning"},
		"additionalProperties": false,
	}
}

// ImagePromptSchema returns the JSON schema for image prompt generation output.
func ImagePromptSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"image_prompt": map[string]any{"type": "string"},
		},
		"required":             []string{"image_prompt"},
		"additionalProperties": false,
	}
}

// CompeteResultsSchema returns the JSON schema for content competition ranking results.
// Rankings are wrapped in {"rankings": [...]}.
func CompeteResultsSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"rankings": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"content_id": map[string]any{"type": "integer"},
						"rank":       map[string]any{"type": "integer"},
						"score":      map[string]any{"type": "number"},
						"reasoning":  map[string]any{"type": "string"},
					},
					"required":             []string{"content_id", "rank", "score", "reasoning"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"rankings"},
		"additionalProperties": false,
	}
}
