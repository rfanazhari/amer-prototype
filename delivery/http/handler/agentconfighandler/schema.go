package agentconfighandler

// hrConfigurationSchema documents approved HR field bounds and empty-draft defaults.
func hrConfigurationSchema() map[string]any {
	return map[string]any{
		"jobDescriptionRef":  map[string]any{"type": "stableRef", "nullable": true, "default": nil},
		"mandatoryQuestions": map[string]any{"type": "array", "minItems": 0, "maxItems": 30, "default": []any{}},
		"competencyTopics":   map[string]any{"type": "array", "minItems": 0, "maxItems": 20, "default": []any{}},
		"stageSeconds":       map[string]any{"type": "object", "nullable": true, "required": []string{"background", "screening", "compensation", "closing"}, "default": nil},
		"compensationPolicy": map[string]any{"type": "object", "nullable": true, "writeOnly": []string{"salaryCeilingRef"}, "default": nil},
		"publicationSeconds": map[string]any{"minTotal": 300, "maxTotal": 600, "questionCapacitySeconds": 30},
	}
}

// documentConfigurationSchema documents the five catalog invoice fields and empty default.
func documentConfigurationSchema() map[string]any {
	return map[string]any{
		"requiredFields": map[string]any{
			"type": "array", "minItems": 0, "maxItems": 5, "default": []any{},
			"enum": []string{"invoice_number", "vendor", "date", "total", "currency"},
		},
		"warningPolicy": map[string]any{"code": "CURRENCY_NOT_REQUIRED", "when": "total without currency"},
	}
}
