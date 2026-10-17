output "environment_id" {
  description = "ID of the throwaway environment created by this apply."
  value       = pingone_environment.this.id
}
