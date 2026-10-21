output "sign_on_policy_id" {
  description = "ID of the sign-on policy (consumed by the application sign-on-policy-assignment fixture)."
  value       = pingone_sign_on_policy.e2e.id
}
