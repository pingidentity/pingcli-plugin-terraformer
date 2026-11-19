output "application_id" {
  description = "ID of the application (consumed by the sign-on-policy-assignment and attribute-mapping fixtures)."
  value       = pingone_application.e2e.id
}
