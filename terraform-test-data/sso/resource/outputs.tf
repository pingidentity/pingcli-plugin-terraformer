output "resource_id" {
  description = "ID of the custom resource (consumed by the resource-scope and resource-attribute fixtures)."
  value       = pingone_resource.e2e.id
}
