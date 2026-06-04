package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// resource is the fetch result for a single endpoint.
type resource struct {
	URI   string          `json:"uri"`
	Body  json.RawMessage `json:"body,omitempty"`
	Error string          `json:"_error,omitempty"`
}

// item is a queued URI together with its crawl depth.
type item struct {
	uri   string
	depth int
}

// crawler follows @odata.id references recursively to collect resources.
type crawler struct {
	c          *client
	maxDepth   int
	visited    map[string]bool
	results    []resource
	errorCount int
}

func newCrawler(c *client, maxDepth int) *crawler {
	return &crawler{
		c:        c,
		maxDepth: maxDepth,
		visited:  make(map[string]bool),
	}
}

// run crawls breadth-first from the entry point.
// Requests run serially, with a random sleep after each one.
func (cr *crawler) run(ctx context.Context, entry string) {
	queue := []item{{normalize(entry), 0}}

	for len(queue) > 0 {
		if ctx.Err() != nil {
			fmt.Fprintln(os.Stderr, "interrupted")
			return
		}

		cur := queue[0]
		queue = queue[1:]

		if cr.visited[cur.uri] {
			continue
		}
		cr.visited[cur.uri] = true

		fmt.Fprintf(os.Stderr, "[depth %d] GET %s\n", cur.depth, cur.uri)

		body, ok := cr.fetch(ctx, cur.uri)
		queue = append(queue, cr.enqueueChildren(cur, body, ok)...)

		cr.c.sleep(ctx)
	}
}

// fetch retrieves one resource, records it, and reports whether it succeeded.
func (cr *crawler) fetch(ctx context.Context, uri string) ([]byte, bool) {
	body, err := cr.c.get(ctx, uri)
	res := resource{URI: uri}
	if err != nil {
		res.Error = err.Error()
		cr.errorCount++
		cr.results = append(cr.results, res)
		return nil, false
	}
	res.Body = json.RawMessage(body)
	cr.results = append(cr.results, res)
	return body, true
}

// enqueueChildren returns the child links to crawl next.
// It returns nothing when the fetch failed or the depth limit is reached.
func (cr *crawler) enqueueChildren(cur item, body []byte, ok bool) []item {
	if !ok {
		return nil
	}
	if cr.maxDepth != 0 && cur.depth >= cr.maxDepth {
		return nil
	}

	var next []item
	for _, link := range extractODataIDs(body) {
		link = normalize(link)
		if link == "" || cr.visited[link] {
			continue
		}
		next = append(next, item{link, cur.depth + 1})
	}
	return next
}

// extractODataIDs collects every "@odata.id" value in the JSON body recursively.
// It covers Members, Links, and any nested NavigationProperty.
func extractODataIDs(body []byte) []string {
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return nil
	}
	seen := make(map[string]bool)
	collectODataIDs(v, seen)

	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// collectODataIDs walks n recursively and records every "@odata.id" string into seen.
func collectODataIDs(n any, seen map[string]bool) {
	switch t := n.(type) {
	case map[string]any:
		for k, val := range t {
			if k != "@odata.id" {
				collectODataIDs(val, seen)
				continue
			}
			if s, ok := val.(string); ok {
				seen[s] = true
			}
		}
	case []any:
		for _, e := range t {
			collectODataIDs(e, seen)
		}
	}
}

// normalize reduces a URI to a same-origin Redfish path.
// It drops external URLs and fragments ($ref) and preserves the trailing slash.
func normalize(uri string) string {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return ""
	}
	// Drop the fragment (for example the #/... of a $ref).
	if i := strings.IndexByte(uri, '#'); i >= 0 {
		uri = uri[:i]
	}
	// Allow same-origin absolute paths only.
	if !strings.HasPrefix(uri, "/redfish/") {
		return ""
	}
	return uri
}
