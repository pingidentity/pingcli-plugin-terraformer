package core

// Live-data label verification for #166. Skipped unless the file exists:
//   go test ./internal/core/ -run TestLiveVariableLabels -v
// with PINGCLI_TERRAFORMER_LIVE_VARS_JSON pointing at the JSON body of
// GET /environments/{id}/variables (fetched via pingcli). This runs the
// REAL embedded variable.yaml definition + processor + assignLabels against
// genuine API response data without needing worker credentials.

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/definitions"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/pingidentity/pingone-go-client/pingone"
	"github.com/stretchr/testify/require"
)

func TestLiveVariableLabels(t *testing.T) {
	path := os.Getenv("PINGCLI_TERRAFORMER_LIVE_VARS_JSON")
	if path == "" {
		t.Skip("set PINGCLI_TERRAFORMER_LIVE_VARS_JSON to a live GET /variables response body")
	}
	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var body struct {
		Data struct {
			Embedded struct {
				Variables []*pingone.DaVinciVariableResponse `json:"variables"`
			} `json:"_embedded"`
		} `json:"data"`
	}

	require.NoError(t, json.Unmarshal(raw, &body))
	require.NotEmpty(t, body.Data.Embedded.Variables)

	reg := schema.NewRegistry()
	require.NoError(t, reg.LoadFromFS(definitions.FS, "pingone"))

	proc := NewProcessor(reg)
	items := make([]interface{}, len(body.Data.Embedded.Variables))
	for i, v := range body.Data.Embedded.Variables {
		items[i] = v // SDK struct — processor reflects Go field names (Id, Flow.Id, ...)
	}
	processed, err := proc.ProcessResourceList("pingone_davinci_variable", items)
	require.NoError(t, err)

	def, err := reg.Get("pingone_davinci_variable")
	require.NoError(t, err)

	// Feed each processed resource its own ID for deterministic suffix order.
	for i, p := range processed {
		p.ID = body.Data.Embedded.Variables[i].Id.String()
	}
	err = assignLabels(processed, def)
	t.Logf("assignLabels warning/err (nil = no collisions): %v", err)

	seen := map[string]int{}
	for _, p := range processed {
		t.Logf("label: %-60s (name=%s context=%s)", p.Label, p.Attributes["name"], p.Attributes["context"])
		seen[p.Label]++
	}
	for label, n := range seen {
		require.Equal(t, 1, n, "label %s assigned to %d resources", label, n)
	}
}
