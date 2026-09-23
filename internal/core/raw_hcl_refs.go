package core

import "strings"

// bareRefSentinel marks a quoted, JSON-safe placeholder standing in for a bare
// Terraform traversal token inside a RawHCLValue jsonencode(...) blob.
//
// Bare tokens (var.X, type.label.field) are not valid JSON, so substituting
// one directly would break any later re-parse of the blob — for example the
// next embedded-reference rule application, which unmarshals the same blob
// (see processRawHCLValue). Producers therefore embed the quoted sentinel
// form while the blob must remain valid JSON, and a final materialization
// sweep swaps each quoted placeholder for the bare token once all blob
// rewriting is done.
const bareRefSentinel = "@@TFREF@@"

// BareRefPlaceholder returns a quoted, JSON-parseable sentinel string that
// stands in for the bare Terraform traversal token (e.g. "var.my_var" or
// "pingone_davinci_flow.my_label.id"). The token must draw only from
// [A-Za-z0-9_.-] — Terraform traversals are built from identifier segments
// and dots, and variable names are sanitized to [a-zA-Z0-9_] by
// utils.SanitizeVariableName — so the token never contains quotes,
// backslashes, or control characters, and the quoted result is a valid JSON
// string with no escaping. Producers writing directly into blob text use
// this quoted form; producers whose value passes through encoding/json
// (e.g. json.MarshalIndent) must use BareRefSentinelCore instead — the
// marshaler supplies the surrounding quotes, and this quoted form would be
// double-encoded.
func BareRefPlaceholder(token string) string {
	return `"` + bareRefSentinel + token + bareRefSentinel + `"`
}

// BareRefSentinelCore returns the unquoted sentinel marker wrapping token:
// @@TFREF@@<token>@@. It is the marshal-safe form for producers that store a
// value into a Go map which encoding/json later marshals (e.g.
// json.MarshalIndent in a blob builder): the marshaler supplies the
// surrounding quotes, so the marshaled text contains the full quoted
// placeholder BareRefPlaceholder(token) — wrapping BareRefPlaceholder's
// already-quoted form in a marshal pass would double-encode its quotes and
// defeat MaterializeBareRefs.
func BareRefSentinelCore(token string) string {
	return bareRefSentinel + token + bareRefSentinel
}

// MaterializeBareRefs replaces every quoted bare-ref placeholder in blob with
// its bare traversal token (quotes dropped), producing final HCL text.
// Placeholders have the exact shape `"@@TFREF@@<token>@@"`; the token itself
// contains no quotes, so the first `@@"` after the opening marker closes the
// placeholder. Blobs containing no placeholder opening marker are returned
// unchanged. A bare @@TFREF@@ appearing inside unrelated string data is left
// as-is — only the full quoted placeholder form is rewritten. Residual risk
// accepted by design: a string value that happens to be exactly
// `"@@TFREF@@<token>@@"` (token from [A-Za-z0-9_.-]) would be rewritten; real
// PingOne data cannot produce that form because both sanitizers eliminate '@'
// from generated tokens and no producer emits the marker into user-visible
// strings.
func MaterializeBareRefs(blob string) string {
	prefix := `"` + bareRefSentinel
	suffix := bareRefSentinel + `"`

	for {
		start := strings.Index(blob, prefix)
		if start < 0 {
			return blob
		}

		tokenStart := start + len(prefix)
		end := strings.Index(blob[tokenStart:], suffix)
		if end < 0 {
			// Unterminated placeholder — leave the text untouched rather than
			// corrupting the blob.
			return blob
		}

		token := blob[tokenStart : tokenStart+end]
		blob = blob[:start] + token + blob[tokenStart+end+len(suffix):]
	}
}

// materializeBareRefsInAttributes walks an attributes tree (maps, slices, and
// RawHCLValue leaves, mirroring walkAndProcessPath's traversal) and swaps
// every quoted bare-ref placeholder in RawHCLValue blobs for its bare token.
// Non-blob values are returned unchanged. The walk mutates maps and slices in
// place, matching the convention used by the embedded-reference processors.
func materializeBareRefsInAttributes(value interface{}) interface{} {
	switch typed := value.(type) {
	case RawHCLValue:
		return RawHCLValue(MaterializeBareRefs(string(typed)))
	case map[string]interface{}:
		for key, val := range typed {
			typed[key] = materializeBareRefsInAttributes(val)
		}
		return typed
	case []interface{}:
		for i, val := range typed {
			typed[i] = materializeBareRefsInAttributes(val)
		}
		return typed
	default:
		return value
	}
}
