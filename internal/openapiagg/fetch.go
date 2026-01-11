package openapiagg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

const (
	maxOpenAPISize   int64 = 5 << 20 // 5MB safety cap
	defaultHTTPDelay       = 8 * time.Second
)

type fetchedSpec struct {
	Source Source
	Doc    *openapi3.T
	Raw    []byte
}

func newHTTPClient() *http.Client {
	return &http.Client{
		Timeout: defaultHTTPDelay,
	}
}

func fetchSpec(ctx context.Context, client *http.Client, src Source) (*fetchedSpec, error) {
	u, err := url.Parse(src.OpenAPIURL)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	var raw []byte
	switch strings.ToLower(u.Scheme) {
	case "file":
		raw, err = os.ReadFile(u.Path)
		if err != nil {
			return nil, fmt.Errorf("read file: %w", err)
		}
	default:
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, src.OpenAPIURL, nil)
		if reqErr != nil {
			return nil, fmt.Errorf("build request: %w", reqErr)
		}
		req.Header.Set("Accept", "application/json, application/yaml, */*")

		resp, doErr := client.Do(req)
		if doErr != nil {
			return nil, fmt.Errorf("fetch: %w", doErr)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
		}

		limited := io.LimitReader(resp.Body, maxOpenAPISize)
		raw, err = io.ReadAll(limited)
		if err != nil {
			return nil, fmt.Errorf("read body: %w", err)
		}
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false

	doc, err := loader.LoadFromData(raw)
	if err != nil {
		return nil, fmt.Errorf("parse openapi: %w", err)
	}

	return &fetchedSpec{
		Source: src,
		Doc:    doc,
		Raw:    raw,
	}, nil
}
