variable "environment_id" {
  description = "ID of the environment containing both groups."
  type        = string
}

variable "parent_group_id" {
  description = "ID of the parent group."
  type        = string
}

variable "child_group_id" {
  description = "ID of the group to nest under the parent."
  type        = string
}

resource "pingone_group_nesting" "e2e" {
  environment_id  = var.environment_id
  group_id        = var.parent_group_id
  nested_group_id = var.child_group_id
}
