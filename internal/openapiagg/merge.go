package openapiagg

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

type mergeIssue struct {
	Source string `json:"source"`
	Detail string `json:"detail"`
}

type mergeInput struct {
	Source Source
	Doc    *openapi3.T
}

func mergeSpecs(inputs []mergeInput) (*openapi3.T, []mergeIssue, error) {
	if len(inputs) == 0 {
		return nil, nil, errors.New("no specs to merge")
	}

	agg := &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:       "Aggregated OpenAPI",
			Description: "Merged by openapi-aggregator; paths are prefixed per service",
			Version:     time.Now().UTC().Format(time.RFC3339),
		},
		Paths: openapi3.NewPaths(),
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{},
		},
		Servers: []*openapi3.Server{
			{URL: "/"},
		},
	}

	var issues []mergeIssue
	operationIDs := map[string]struct{}{}

	for _, input := range inputs {
		if input.Doc == nil {
			issues = append(issues, mergeIssue{Source: input.Source.Name, Detail: "missing spec doc"})
			continue
		}

		preparedDoc, prepIssues, err := prepareForMerge(input, operationIDs)
		if err != nil {
			issues = append(issues, mergeIssue{Source: input.Source.Name, Detail: err.Error()})
			continue
		}
		issues = append(issues, prepIssues...)

		// components (schemas only for now)
		for name, schema := range preparedDoc.Components.Schemas {
			if _, exists := agg.Components.Schemas[name]; exists {
				issues = append(issues, mergeIssue{
					Source: input.Source.Name,
					Detail: fmt.Sprintf("schema collision on %s", name),
				})
				continue
			}
			agg.Components.Schemas[name] = schema
		}

		// paths
		for path, item := range preparedDoc.Paths.Map() {
			if existing := agg.Paths.Value(path); existing != nil {
				issues = append(issues, mergeIssue{
					Source: input.Source.Name,
					Detail: fmt.Sprintf("path collision on %s", path),
				})
				continue
			}
			agg.Paths.Set(path, item)
		}

		if len(preparedDoc.Tags) > 0 {
			agg.Tags = append(agg.Tags, preparedDoc.Tags...)
		}
	}

	if agg.Paths == nil || agg.Paths.Len() == 0 {
		return nil, issues, errors.New("no paths merged")
	}

	return agg, issues, nil
}

func prepareForMerge(input mergeInput, opIndex map[string]struct{}) (*openapi3.T, []mergeIssue, error) {
	doc, err := cloneDoc(input.Doc)
	if err != nil {
		return nil, nil, err
	}
	var issues []mergeIssue

	componentPrefix := input.Source.ComponentPrefixSafe()
	doc, err = prefixComponents(doc, componentPrefix)
	if err != nil {
		return nil, nil, err
	}

	pathPrefix := normalizePrefix(input.Source.PathPrefix)
	doc.Paths = prefixPaths(doc.Paths, pathPrefix)

	opPrefix := sanitizeIdentifier(input.Source.OperationPrefixSafe())
	issues = append(issues, ensureOperationIDs(doc.Paths, opPrefix, input.Source.Name, opIndex)...)

	if doc.Tags == nil {
		doc.Tags = openapi3.Tags{}
	}
	doc.Tags = append(doc.Tags, &openapi3.Tag{
		Name:        input.Source.Name,
		Description: fmt.Sprintf("Imported from %s (path prefix: %s)", input.Source.Name, pathPrefix),
	})

	return doc, issues, nil
}

func prefixComponents(doc *openapi3.T, prefix string) (*openapi3.T, error) {
	if doc.Components == nil {
		doc.Components = &openapi3.Components{}
	}

	if doc.Components.Schemas == nil || len(doc.Components.Schemas) == 0 {
		return doc, nil
	}
	mapping := map[string]string{}
	renamed := make(map[string]*openapi3.SchemaRef, len(doc.Components.Schemas))

	for name, schema := range doc.Components.Schemas {
		newName := prefix + "__" + name
		if _, exists := renamed[newName]; exists {
			return nil, fmt.Errorf("duplicate schema after prefix: %s", newName)
		}
		renamed[newName] = schema
		mapping["#/components/schemas/"+name] = "#/components/schemas/" + newName
	}

	doc.Components.Schemas = renamed
	if len(mapping) == 0 {
		return doc, nil
	}

	data, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	for oldRef, newRef := range mapping {
		data = bytes.ReplaceAll(data, []byte(oldRef), []byte(newRef))
	}

	var out openapi3.T
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func prefixPaths(paths *openapi3.Paths, prefix string) *openapi3.Paths {
	if paths == nil {
		return openapi3.NewPaths()
	}
	if prefix == "" || prefix == "/" {
		return paths
	}
	out := openapi3.NewPaths()
	for path, item := range paths.Map() {
		out.Set(joinPath(prefix, path), item)
	}
	return out
}

func ensureOperationIDs(paths *openapi3.Paths, prefix, source string, global map[string]struct{}) []mergeIssue {
	var issues []mergeIssue

	apply := func(method string, op *openapi3.Operation, path string) {
		if op == nil {
			return
		}
		base := op.OperationID
		if base == "" {
			base = fmt.Sprintf("%s_%s_%s", prefix, strings.ToLower(method), sanitizePath(path))
		}
		if !strings.HasPrefix(base, prefix) {
			base = fmt.Sprintf("%s_%s", prefix, base)
		}
		id := base
		suffix := 1
		for {
			if _, exists := global[id]; !exists {
				global[id] = struct{}{}
				op.OperationID = id
				return
			}
			issues = append(issues, mergeIssue{
				Source: source,
				Detail: fmt.Sprintf("operationId collision on %s, renaming", id),
			})
			suffix++
			id = fmt.Sprintf("%s_%d", base, suffix)
		}
	}

	for path, item := range paths.Map() {
		apply("get", item.Get, path)
		apply("put", item.Put, path)
		apply("post", item.Post, path)
		apply("delete", item.Delete, path)
		apply("options", item.Options, path)
		apply("head", item.Head, path)
		apply("patch", item.Patch, path)
		apply("trace", item.Trace, path)
	}

	return issues
}

func cloneDoc(doc *openapi3.T) (*openapi3.T, error) {
	data, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	var out openapi3.T
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func normalizePrefix(in string) string {
	if in == "" {
		return ""
	}
	if !strings.HasPrefix(in, "/") {
		in = "/" + in
	}
	in = strings.TrimSuffix(in, "/")
	if in == "" {
		return "/"
	}
	return in
}

func joinPath(prefix, path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if prefix == "/" {
		return path
	}
	return prefix + path
}

func sanitizePath(p string) string {
	p = strings.ReplaceAll(p, "/", "_")
	p = strings.ReplaceAll(p, "{", "")
	p = strings.ReplaceAll(p, "}", "")
	p = strings.ReplaceAll(p, "-", "_")
	return strings.Trim(p, "_")
}

func sanitizeIdentifier(in string) string {
	in = strings.ReplaceAll(in, "-", "_")
	in = strings.ReplaceAll(in, " ", "_")
	return in
}
