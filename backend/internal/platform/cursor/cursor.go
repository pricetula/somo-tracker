package cursor

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Cursor is the decoded payload of a pagination cursor.
type Cursor struct {
	Sort       []string      `json:"s"`  // sort names bound to this cursor
	Last       []interface{} `json:"l"`  // last row values for each sort column
	Tiebreaker string        `json:"t"`  // UUID PK of last row
	Ctx        string        `json:"c"`  // query-context hash
}

// EncodeCursor encodes a cursor payload into a Base64url string.
func EncodeCursor(payload *Cursor) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}
	return base64.URLEncoding.WithPadding(-1).EncodeToString(data), nil
}

// DecodeCursor decodes a Base64url cursor string into a Cursor payload.
func DecodeCursor(token string) (*Cursor, error) {
	data, err := base64.URLEncoding.WithPadding(-1).DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}
	var c Cursor
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("unmarshal cursor: %w", err)
	}
	return &c, nil
}

// ContextHash computes the SHA-256 hash of the sort + filters context.
// This binds a cursor to the exact sort and filter set that produced it.
func ContextHash(sortName string, filters map[string]FilterValue) string {
	// Canonicalize filters: sort by field name, then serialize op:value pairs
	keys := make([]string, 0, len(filters))
	for k := range filters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	parts = append(parts, sortName)
	for _, k := range keys {
		fv := filters[k]
		parts = append(parts, fmt.Sprintf("%s=%v:%v", k, fv.Op, fv.Val))
	}
	input := strings.Join(parts, "|")

	h := sha256.Sum256([]byte(input))
	return base64.URLEncoding.WithPadding(-1).EncodeToString(h[:])
}

// ContextHashMatches checks if the cursor's context hash matches the current
// sort + filter set.
func (c *Cursor) ContextHashMatches(sortName string, filters map[string]FilterValue) bool {
	return c.Ctx == ContextHash(sortName, filters)
}
