output "parent_group_id" {
  description = "ID of the parent group (consumed by the group-nesting fixture)."
  value       = pingone_group.e2e.id
}

output "child_group_id" {
  description = "ID of the child group (consumed by the group-nesting fixture)."
  value       = pingone_group.e2e_child.id
}
