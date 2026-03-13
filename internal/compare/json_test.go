package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareJSON_Identical(t *testing.T) {
	j := `{
  "resource": {
    "pingone_davinci_variable": {
      "my_var": {
        "name": "myVar",
        "environment_id": "env-123"
      }
    }
  }
}`
	result, err := CompareJSON(j, j)
	require.NoError(t, err)
	assert.False(t, result.HasDiffs())
}

func TestCompareJSON_MissingResource(t *testing.T) {
	expected := `{
  "resource": {
    "pingone_davinci_variable": {
      "var_a": { "name": "a" },
      "var_b": { "name": "b" }
    }
  }
}`
	actual := `{
  "resource": {
    "pingone_davinci_variable": {
      "var_a": { "name": "a" }
    }
  }
}`
	result, err := CompareJSON(expected, actual)
	require.NoError(t, err)
	assert.True(t, result.HasDiffs())
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffMissingResource, result.Diffs[0].Kind)
	assert.Equal(t, "pingone_davinci_variable.var_b", result.Diffs[0].Resource)
}

func TestCompareJSON_ExtraResource(t *testing.T) {
	expected := `{
  "resource": {
    "pingone_davinci_variable": {
      "var_a": { "name": "a" }
    }
  }
}`
	actual := `{
  "resource": {
    "pingone_davinci_variable": {
      "var_a": { "name": "a" },
      "var_extra": { "name": "extra" }
    }
  }
}`
	result, err := CompareJSON(expected, actual)
	require.NoError(t, err)
	assert.True(t, result.HasDiffs())
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffExtraResource, result.Diffs[0].Kind)
	assert.Equal(t, "pingone_davinci_variable.var_extra", result.Diffs[0].Resource)
}

func TestCompareJSON_ValueMismatch(t *testing.T) {
	expected := `{
  "resource": {
    "pingone_davinci_variable": {
      "v": { "name": "old_name" }
    }
  }
}`
	actual := `{
  "resource": {
    "pingone_davinci_variable": {
      "v": { "name": "new_name" }
    }
  }
}`
	result, err := CompareJSON(expected, actual)
	require.NoError(t, err)
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffValueMismatch, result.Diffs[0].Kind)
	assert.Equal(t, "name", result.Diffs[0].Attribute)
}

func TestCompareJSON_MissingAttribute(t *testing.T) {
	expected := `{
  "resource": {
    "pingone_davinci_variable": {
      "v": { "name": "test", "description": "desc" }
    }
  }
}`
	actual := `{
  "resource": {
    "pingone_davinci_variable": {
      "v": { "name": "test" }
    }
  }
}`
	result, err := CompareJSON(expected, actual)
	require.NoError(t, err)
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffMissingAttribute, result.Diffs[0].Kind)
	assert.Equal(t, "description", result.Diffs[0].Attribute)
}

func TestCompareJSON_ExtraAttribute(t *testing.T) {
	expected := `{
  "resource": {
    "pingone_davinci_variable": {
      "v": { "name": "test" }
    }
  }
}`
	actual := `{
  "resource": {
    "pingone_davinci_variable": {
      "v": { "name": "test", "extra": "val" }
    }
  }
}`
	result, err := CompareJSON(expected, actual)
	require.NoError(t, err)
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffExtraAttribute, result.Diffs[0].Kind)
	assert.Equal(t, "extra", result.Diffs[0].Attribute)
}

func TestCompareJSON_MultipleResourceTypes(t *testing.T) {
	expected := `{
  "resource": {
    "pingone_davinci_variable": {
      "v": { "name": "var" }
    },
    "pingone_davinci_flow": {
      "f": { "name": "flow" }
    }
  }
}`
	result, err := CompareJSON(expected, expected)
	require.NoError(t, err)
	assert.False(t, result.HasDiffs())
}

func TestCompareJSON_EmptyInputs(t *testing.T) {
	result, err := CompareJSON("{}", "{}")
	require.NoError(t, err)
	assert.False(t, result.HasDiffs())
}

func TestCompareJSON_InvalidJSON_Expected(t *testing.T) {
	_, err := CompareJSON("{invalid", "{}")
	require.Error(t, err)
}

func TestCompareJSON_InvalidJSON_Actual(t *testing.T) {
	_, err := CompareJSON("{}", "{invalid")
	require.Error(t, err)
}

func TestCompareJSON_MissingResourceType(t *testing.T) {
	expected := `{
  "resource": {
    "pingone_davinci_variable": {
      "v": { "name": "test" }
    },
    "pingone_davinci_flow": {
      "f": { "name": "flow" }
    }
  }
}`
	actual := `{
  "resource": {
    "pingone_davinci_variable": {
      "v": { "name": "test" }
    }
  }
}`
	result, err := CompareJSON(expected, actual)
	require.NoError(t, err)
	assert.True(t, result.HasDiffs())
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffMissingResource, result.Diffs[0].Kind)
	assert.Equal(t, "pingone_davinci_flow.f", result.Diffs[0].Resource)
}

func TestCompareJSON_ExtraResourceType(t *testing.T) {
	expected := `{
  "resource": {
    "pingone_davinci_variable": {
      "v": { "name": "test" }
    }
  }
}`
	actual := `{
  "resource": {
    "pingone_davinci_variable": {
      "v": { "name": "test" }
    },
    "pingone_davinci_flow": {
      "f": { "name": "flow" }
    }
  }
}`
	result, err := CompareJSON(expected, actual)
	require.NoError(t, err)
	assert.True(t, result.HasDiffs())
	require.Len(t, result.Diffs, 1)
	assert.Equal(t, DiffExtraResource, result.Diffs[0].Kind)
	assert.Equal(t, "pingone_davinci_flow.f", result.Diffs[0].Resource)
}

func TestCompareJSON_NestedObjectAttribute(t *testing.T) {
	expected := `{
  "resource": {
    "pingone_davinci_flow": {
      "f": {
        "name": "flow",
        "settings": { "key": "old_value" }
      }
    }
  }
}`
	actual := `{
  "resource": {
    "pingone_davinci_flow": {
      "f": {
        "name": "flow",
        "settings": { "key": "new_value" }
      }
    }
  }
}`
	result, err := CompareJSON(expected, actual)
	require.NoError(t, err)
	assert.True(t, result.HasDiffs())
	// Nested object differences should be reported
	found := false
	for _, d := range result.Diffs {
		if d.Kind == DiffValueMismatch || d.Kind == DiffBlockMismatch {
			found = true
		}
	}
	assert.True(t, found, "should detect nested object differences")
}
