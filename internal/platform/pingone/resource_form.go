package pingone

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/patrickcping/pingone-go-sdk-v2/management"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

// pingone_form manages DaVinci-style sign-on forms through the PingOne
// platform FormManagementApi. Environment-scoped: a plain paginated list.
//
// The components.fields union (24 FormField variants) is projected to a
// generic JSON map preserving every variant's declared fields, so exports
// round-trip variant-specific data (options, icons, polling config) the
// provider models as variant-specific schema. Slate text (label, content)
// is an escaped-JSON string in the API; the projection keeps it verbatim
// and the emitted HCL jsonencode()s it, matching the provider's examples.
func init() {
	registerResource("pingone_form", resourceHandler{
		list: listForms,
		get:  getForm,
	})

	registerTransform("handleFormFieldProjection", handleFormFieldProjection)
}

// listForms lists all forms in the environment, then re-reads each with
// include=components (list-then-get): the list endpoint returns
// components.fields as null, and only the single-get with the include
// parameter populates the field union the projection needs.
func listForms(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	var ids []string
	iterator := mgmt.FormManagementApi.ReadAllForms(ctx, c.environmentID.String()).Execute()
	for cursor, err := range iterator {
		if err != nil {
			return nil, fmt.Errorf("list forms: %w", err)
		}
		embedded, ok := cursor.EntityArray.GetEmbeddedOk()
		if !ok || embedded == nil {
			continue
		}
		forms, ok := embedded.GetFormsOk()
		if !ok {
			continue
		}
		for i := range forms {
			if id := forms[i].GetId(); id != "" {
				ids = append(ids, id)
			}
		}
	}

	var result []interface{}
	for _, id := range ids {
		form, err := getForm(ctx, c, "", id)
		if err != nil {
			return nil, err
		}
		result = append(result, form)
	}
	return result, nil
}

// getForm retrieves a single form with its components included.
func getForm(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	form, _, err := mgmt.FormManagementApi.ReadForm(ctx, c.environmentID.String(), resourceID).
		Include(management.ENUMFORMSINCLUDEPARAMETER_COMPONENTS).Execute()
	if err != nil {
		return nil, fmt.Errorf("get form: %w", err)
	}
	return form, nil
}

// handleFormFieldProjection converts the FormComponents union list into the
// provider's components HCL object: components = { fields = [ ... ] }, where
// each field carries type/position/key/label/etc. Slate text (label,
// content) is an escaped-JSON string in the API; the provider's examples
// wrap those in jsonencode(...), so the projection emits them as
// jsonencode() expressions too.
func handleFormFieldProjection(value interface{}, apiData interface{}, _ *schema.AttributeDefinition, _ *schema.ResourceDefinition) (interface{}, error) {
	// convertValue passes object attributes through as the struct value
	// (management.FormComponents, not a pointer).
	var components *management.FormComponents
	switch v := value.(type) {
	case *management.FormComponents:
		components = v
	case management.FormComponents:
		components = &v
	default:
		return nil, nil
	}
	if components == nil {
		return nil, nil
	}

	fields := components.GetFields()
	var buf strings.Builder
	buf.WriteString("{\n  fields = [")
	wrote := 0
	for i := range fields {
		m := projectFormField(&fields[i])
		if m == nil {
			continue
		}
		if wrote > 0 {
			buf.WriteString(",")
		}
		wrote++
		buf.WriteString("\n    ")
		if err := writeHCLObject(&buf, m, "    "); err != nil {
			return nil, err
		}
	}
	buf.WriteString("\n  ]\n}")

	// Escape HCL template markers so string literals survive Terraform's
	// interpolation scanner (same treatment as core.transformJSONEncodeRaw).
	s := strings.ReplaceAll(buf.String(), "${", "$${")
	s = strings.ReplaceAll(s, "%{", "%%{")
	return core.RawHCLValue(s), nil
}

// writeHCLObject writes m as an HCL object literal at the given indent.
func writeHCLObject(buf *strings.Builder, m map[string]interface{}, indent string) error {
	buf.WriteString(indent + "{\n")
	keys := make([]string, 0, len(m))
	for k := range m {
		if k == "_links" { // HAL noise never belongs in HCL
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		buf.WriteString(indent + "  " + hclKey(k) + " = ")
		if err := writeHCLValue(buf, m[k], indent+"  "); err != nil {
			return err
		}
		if i < len(keys)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString(indent + "}")
	return nil
}

// hclKey quotes object keys that are not bare HCL identifiers.
func hclKey(k string) string {
	if bareIdentifier.MatchString(k) {
		return k
	}
	return fmt.Sprintf("%q", k)
}

// bareIdentifier matches keys writable as bare HCL object keys.
var bareIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// writeHCLValue writes one value: objects recursively, escaped-JSON slate
// strings re-parsed and emitted as jsonencode(<the JSON object>) — the API
// value IS JSON text, and jsonencode() of the parsed object produces the
// same shape the provider's examples use; marshaling the string instead
// would double-encode it (issue: slate labels rendered mangled in the UI) —
// and everything else as a JSON literal (JSON scalars are valid HCL
// literals; json.Marshal provides the quoting).
func writeHCLValue(buf *strings.Builder, v interface{}, indent string) error {
	switch val := v.(type) {
	case map[string]interface{}:
		if len(val) == 0 {
			buf.WriteString("{}")
			return nil
		}
		return writeHCLObject(buf, val, indent)
	case []interface{}:
		if len(val) == 0 {
			buf.WriteString("[]")
			return nil
		}
		buf.WriteString("[")
		for i, e := range val {
			if i > 0 {
				buf.WriteString(",")
			}
			if err := writeHCLValue(buf, e, indent); err != nil {
				return err
			}
		}
		buf.WriteString("]")
		return nil
	case string:
		if looksLikeSlateJSON(val) {
			// The API value is itself JSON text (Slate rich text: label,
			// content, labelPasswordVerify). Emit it verbatim inside
			// jsonencode(...) so HCL re-parses it into the object the
			// provider stores — jsonencode([{"type":...}]) in the provider's
			// examples. Marshaling the string first (jsonencode("[{\"type..")
			// double-encodes: the provider stores an escaped literal blob and
			// the UI shows mangled text.
			//
			// The API value must parse as JSON for this path; re-marshal the
			// decoded value so the emitted expression is canonical (compacted,
			// no leading whitespace) rather than trusting the raw text's
			// formatting. Anything that fails to parse stays a quoted string.
			var decoded interface{}
			if err := json.Unmarshal([]byte(val), &decoded); err != nil {
				// Unreachable for looksLikeSlateJSON inputs in practice; fall
				// back to the quoted string rather than emitting broken HCL.
				b, mErr := json.Marshal(val)
				if mErr != nil {
					return mErr
				}
				buf.Write(b)
				return nil
			}
			b, err := json.Marshal(decoded)
			if err != nil {
				return err
			}
			buf.WriteString("jsonencode(" + string(b) + ")")
			return nil
		}
		b, err := json.Marshal(val)
		if err != nil {
			return err
		}
		buf.Write(b)
		return nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		buf.Write(b)
		return nil
	}
}

// looksLikeSlateJSON reports whether s starts with { or [ — the slate text
// shape (label, content). Plain strings like "TEXT" stay quoted strings.
func looksLikeSlateJSON(s string) bool {
	t := strings.TrimSpace(s)
	return strings.HasPrefix(t, "{") || strings.HasPrefix(t, "[")
}

// projectFormField unwraps one FormField union member and flattens it to a
// JSON map. The SDK structs marshal the full declared field set, so a
// round-trip through json.Marshal of the unwrapped variant preserves every
// variant-specific property without enumerating all 24 types by hand.
// The "type" discriminator rides along, giving the emitted HCL the same
// variant-tagged shape the provider reads.
func projectFormField(field *management.FormField) map[string]interface{} {
	if field == nil {
		return nil
	}
	instance := field.GetActualInstance()
	if instance == nil {
		return nil
	}
	b, err := json.Marshal(instance)
	if err != nil {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	return m
}
