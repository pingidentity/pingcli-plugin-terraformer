# PingOne Provider Resource Coverage

Tracks which resources from the [`pingidentity/pingone`](https://registry.terraform.io/providers/pingidentity/pingone/latest/docs) Terraform provider (v1.19.1, 105 resources total) are supported by this exporter.

Legend: ✅ Supported · ❌ Not yet supported

---

## Platform (41 resources)

| Resource | Supported |
|---|---|
| `pingone_administrator_security` | ❌ |
| `pingone_agreement` | ❌ |
| `pingone_agreement_enable` | ❌ |
| `pingone_agreement_localization` | ❌ |
| `pingone_agreement_localization_enable` | ❌ |
| `pingone_agreement_localization_revision` | ❌ |
| `pingone_alert_channel` | ❌ |
| `pingone_branding_settings` | ❌ |
| `pingone_branding_theme` | ❌ |
| `pingone_branding_theme_default` | ❌ |
| `pingone_certificate` | ❌ |
| `pingone_certificate_signing_response` | ❌ |
| `pingone_custom_domain` | ❌ |
| `pingone_custom_domain_ssl` | ❌ |
| `pingone_custom_domain_verify` | ❌ |
| `pingone_custom_role` | ❌ |
| `pingone_environment` | ✅ |
| `pingone_form` | ❌ |
| `pingone_forms_recaptcha_v2` | ❌ |
| `pingone_gateway` | ❌ |
| `pingone_gateway_credential` | ❌ |
| `pingone_gateway_role_assignment` | ❌ |
| `pingone_identity_propagation_plan` | ❌ |
| `pingone_image` | ❌ |
| `pingone_key` | ❌ |
| `pingone_key_rotation_policy` | ❌ |
| `pingone_language` | ❌ |
| `pingone_language_translation` | ❌ |
| `pingone_language_update` | ❌ |
| `pingone_notification_policy` | ❌ |
| `pingone_notification_settings` | ❌ |
| `pingone_notification_settings_email` | ❌ |
| `pingone_notification_template_content` | ❌ |
| `pingone_phone_delivery_settings` | ❌ |
| `pingone_rate_limit_configuration` | ❌ |
| `pingone_role_assignment_user` | ❌ |
| `pingone_system_application` | ❌ |
| `pingone_trusted_email_address` | ❌ |
| `pingone_trusted_email_domain` | ❌ |
| `pingone_user_role_assignment` | ❌ |
| `pingone_webhook` | ❌ |

## SSO (29 resources)

| Resource | Supported |
|---|---|
| `pingone_application` | ❌ |
| `pingone_application_attribute_mapping` | ❌ |
| `pingone_application_flow_policy_assignment` | ❌ |
| `pingone_application_resource` | ❌ |
| `pingone_application_resource_grant` | ❌ |
| `pingone_application_role_assignment` | ❌ |
| `pingone_application_secret` | ❌ |
| `pingone_application_sign_on_policy_assignment` | ❌ |
| `pingone_group` | ❌ |
| `pingone_group_nesting` | ❌ |
| `pingone_group_role_assignment` | ❌ |
| `pingone_identity_provider` | ❌ |
| `pingone_identity_provider_attribute` | ❌ |
| `pingone_password_policy` | ❌ |
| `pingone_population` | ❌ |
| `pingone_population_default` | ❌ |
| `pingone_population_default_identity_provider` | ❌ |
| `pingone_resource` | ❌ |
| `pingone_resource_attribute` | ❌ |
| `pingone_resource_scope` | ❌ |
| `pingone_resource_scope_openid` | ❌ |
| `pingone_resource_scope_pingone_api` | ❌ |
| `pingone_resource_secret` | ❌ |
| `pingone_schema_attribute` | ❌ |
| `pingone_sign_on_policy` | ❌ |
| `pingone_sign_on_policy_action` | ❌ |
| `pingone_user` | ❌ |
| `pingone_user_application_role_assignment` | ❌ |
| `pingone_user_group_assignment` | ❌ |

## Authorize (7 resources)

| Resource | Supported |
|---|---|
| `pingone_application_resource_permission` | ❌ |
| `pingone_authorize_api_service` | ❌ |
| `pingone_authorize_api_service_deployment` | ❌ |
| `pingone_authorize_api_service_operation` | ❌ |
| `pingone_authorize_application_role` | ❌ |
| `pingone_authorize_application_role_permission` | ❌ |
| `pingone_authorize_decision_endpoint` | ❌ |

## MFA (5 resources)

| Resource | Supported |
|---|---|
| `pingone_mfa_application_push_credential` | ❌ |
| `pingone_mfa_device_policy` | ❌ |
| `pingone_mfa_device_policy_default` | ❌ |
| `pingone_mfa_fido2_policy` | ❌ |
| `pingone_mfa_settings` | ❌ |

## DaVinci (9 resources)

| Resource | Supported |
|---|---|
| `pingone_davinci_application` | ✅ |
| `pingone_davinci_application_flow_policy` | ✅ |
| `pingone_davinci_application_key` | ❌ |
| `pingone_davinci_application_secret` | ❌ |
| `pingone_davinci_connector_instance` | ✅ |
| `pingone_davinci_flow` | ✅ |
| `pingone_davinci_flow_deploy` | ✅ |
| `pingone_davinci_flow_enable` | ✅ |
| `pingone_davinci_variable` | ✅ |

## Protect (2 resources)

| Resource | Supported |
|---|---|
| `pingone_risk_policy` | ❌ |
| `pingone_risk_predictor` | ❌ |

## Neo — Verify & Credentials (7 resources)

| Resource | Supported |
|---|---|
| `pingone_credential_issuance_rule` | ❌ |
| `pingone_credential_issuer_profile` | ❌ |
| `pingone_credential_type` | ❌ |
| `pingone_digital_wallet_application` | ❌ |
| `pingone_verify_policy` | ❌ |
| `pingone_verify_voice_phrase` | ❌ |
| `pingone_verify_voice_phrase_content` | ❌ |

---

## Summary

| Category | Provider total | Exporter supported |
|---|---|---|
| Platform | 41 | 1 |
| SSO | 29 | 0 |
| Authorize | 7 | 0 |
| MFA | 5 | 0 |
| DaVinci | 9 | 7 |
| Protect | 2 | 0 |
| Neo (Verify & Credentials) | 7 | 0 |
| **Total** | **100** | **8** |

> Provider source: [registry.terraform.io/providers/pingidentity/pingone](https://registry.terraform.io/providers/pingidentity/pingone/latest/docs) — v1.19.1
