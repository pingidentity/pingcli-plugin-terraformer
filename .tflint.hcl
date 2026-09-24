# tflint configuration for the terraformer exporter.
#
# Purpose: guard against a regression of tflint's
# terraform_deprecated_interpolation rule on exported HCL (issue #158).
# Exported modules previously carried interpolation-only strings like
# "${var.X}" inside jsonencode({...}) blobs, which the rule flags.
#
# Invoked by:
#   make tflint            — runs against tests/testdata/tflint-sample/
#   .github/workflows/regression.yaml — runs against the fresh HCL export
#
# The bundled terraform ruleset is enabled (no --init required for it), and
# the deprecated-interpolation rule is explicitly enabled so the guard
# survives a future change of the ruleset default.
plugin "terraform" {
  enabled = true
}

rule "terraform_deprecated_interpolation" {
  enabled = true
}

# ─── Rules disabled because valid exporter output intentionally violates them ───
#
# These rules fire on well-formed exported modules and would turn the guard
# permanently red for reasons outside this repo's control. Disabling them
# keeps the check focused on the deprecated-interpolation regression; it does
# not mask that rule in any way.

# Exporter resource labels are SanitizeResourceName output (pingcli__X-0020-Y)
# and variable names are SanitizeVariableName output (capitalized segments),
# never snake_case. 285 findings on the regression snapshot module alone.
rule "terraform_naming_convention" {
  enabled = false
}

# The exporter emits per-resource-type files (pingone_davinci_flow.tf, ...)
# plus versions.tf/variables.tf/outputs.tf — there is no main.tf.
rule "terraform_standard_module_structure" {
  enabled = false
}

# Pre-existing exporter quirk: complex connector properties can be dropped
# from the properties blob (e.g. non-scalar customAttributes values) while
# their variables.tf declaration is still emitted, leaving the variable
# unused. Fixing the exporter is out of scope of the tflint guard.
rule "terraform_unused_declarations" {
  enabled = false
}

# This rule walks every nested object expression; on a real flow export
# (~140k-line pingone_davinci_flow.tf) it does not complete in practical
# time (5+ CPU minutes without finishing locally). Unrelated to the
# deprecated-interpolation guard.
rule "terraform_map_duplicate_keys" {
  enabled = false
}
