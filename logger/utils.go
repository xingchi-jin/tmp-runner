package logger

import (
	"context"

	constant "github.com/harness/runner/logger/customhooks"
)

// AddLogLabelsToContext adds labels to the context that will be printed with the inline/local logs
// and sent to the remote logger.
func AddLogLabelsToContext(ctx context.Context, fields map[string]string) context.Context {
	if ctx == nil || fields == nil {
		return ctx
	}

	newLogLabels := make(map[string]interface{})
	newLogLabels[string(constant.InlineLabelsKey)] = fields
	newLogLabels[string(constant.RemoteLabelsKey)] = fields

	if logLabels, ok := ctx.Value(constant.LogLabelsKey).(map[string]interface{}); ok {
		// Merge Inline Labels
		if inlineLabels, ok := logLabels[string(constant.InlineLabelsKey)].(map[string]string); ok {
			newLogLabels[string(constant.InlineLabelsKey)] = MergeLabels(inlineLabels, fields)
		}
		// Merge Remote Labels
		if remoteLabels, ok := logLabels[string(constant.RemoteLabelsKey)].(map[string]string); ok {
			newLogLabels[string(constant.RemoteLabelsKey)] = MergeLabels(remoteLabels, fields)
		}
	}

	newctx := context.WithValue(ctx, constant.LogLabelsKey, newLogLabels)
	return newctx
}

// AddInlineLogLabelsToContext adds labels to the context that will be printed with the inline/local logs
func AddInlineLogLabelsToContext(ctx context.Context, fields map[string]string) context.Context {
	if ctx == nil || fields == nil {
		return ctx
	}

	newLogLabels := make(map[string]interface{})
	newLogLabels[string(constant.InlineLabelsKey)] = fields

	if logLabels, ok := ctx.Value(constant.LogLabelsKey).(map[string]interface{}); ok {
		// Merge Inline Labels
		if inlineLabels, ok := logLabels[string(constant.InlineLabelsKey)].(map[string]string); ok {
			newLogLabels[string(constant.InlineLabelsKey)] = MergeLabels(inlineLabels, fields)
		}
		if remoteLabels, ok := logLabels[string(constant.RemoteLabelsKey)].(map[string]string); ok {
			newLogLabels[string(constant.RemoteLabelsKey)] = remoteLabels
		}
	}

	return context.WithValue(ctx, constant.LogLabelsKey, newLogLabels)
}

func MergeLabels(map1, map2 map[string]string) map[string]string {
	mergedMap := make(map[string]string)
	for key, value := range map1 {
		mergedMap[key] = value
	}
	for key, value := range map2 {
		mergedMap[key] = value
	}

	return mergedMap
}
