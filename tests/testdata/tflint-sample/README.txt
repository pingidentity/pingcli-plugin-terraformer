# Sample module for the tflint guard (issue #158).
#
# Hand-written to mirror the post-fix exporter output form exactly:
#   - connector instance properties = jsonencode({... "value" : var.X ...})
#     with bare Terraform traversals (the fix in resource_connection.go)
#   - flow node graph_data blobs carrying bare
#     pingone_davinci_connector_instance.<label>.id resource references
#     (the fix in embedded_references.go)
#   - a typed var.X reference outside any blob (environment_id)
#
# `make tflint` runs tflint against this module with the repo-root
# .tflint.hcl. If a producer ever regresses to emitting interpolation-only
# "${var.X}" / "${type.label.field}" strings inside jsonencode blobs, add one
# here (or to pingone_davinci_flow.tf) to reproduce the finding before fixing.
