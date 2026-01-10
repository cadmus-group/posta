//go:build tools
// +build tools

// Package tools tracks development tool dependencies that are pinned in go.mod.
package tools

// Import a real package from the module so the CLI dependency is recorded.
import _ "github.com/oapi-codegen/oapi-codegen/v2/pkg/codegen"
