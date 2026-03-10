# Migrating DaVinci Resources to PingOne Provider

The following guide documents the process of migrating Terraform managed resources from the legacy DaVinci provider (`pingidentity/davinci`) to the new DaVinci resources within the PingOne Terraform provider (`pingidentity/pingone`).

> **NOTE:** The legacy DaVinci provider relied on human user credentials and browser-based SSO. The new PingOne Terraform provider uses PingOne worker applications, eliminating the dependency on human credentials. This is significantly more suitable for automation scenarios such as CI/CD pipelines or GitHub Actions workflows.

The goal of this migration process is to move configuration managed by the legacy provider to the PingOne provider while minimizing impact to live infrastructure. This involves avoiding deletion or recreation of resources and ensuring that `terraform apply` results in no functional changes during the migration.

## Prerequisites

* Existing Terraform configuration managed by the legacy DaVinci provider (`pingidentity/davinci`)
* The `pingcli-terraformer` tool installed
* A PingOne worker application with **DaVinci Admin Read Only** role
* Terraform 1.5+ installed

## Step 1: Set Up Authentication

Configure your worker application credentials that will be used by the `pingcli-terraformer` tool. These credentials require the DaVinci Admin role to read the live environment configuration.

```bash
export PINGCLI_PINGONE_ENVIRONMENT_ID="your-environment-id"
export PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID="your-client-id"
export PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET="your-client-secret"
export PINGCLI_PINGONE_REGION_CODE="NA"  # or EU, AP, CA, AU
```

## Step 2: Export Terraform Configuration

Use the `export` command to generate Terraform HCL from your live environment. This command:
- Reads all Terraform-supported resources in the live environment
- Converts API responses to a reusable Terraform module
- Abstracts environment-specific values to variables
- Maps dependencies between resources automatically

For managing an existing environment, use these flags:

```bash
pingcli-terraformer export \
  --include-imports \
  --include-values \
  --pingone-export-environment-id=''
```

### Command Flags Explained

- `--include-imports`: Generates import blocks for each identified resource, enabling you to bring existing infrastructure under Terraform management
- `--include-values`: Produces a `terraform.tfvars` file with actual values for module variables

The export creates a version-control-ready module that can be used standalone or within a larger root module of configuration.

## Step 3: Prepare for Import

Before running `terraform apply`, you need to handle secret attributes and import existing resources.

### 3.0 Configure Provider

The export will include a versions.tf file that points to the expected PingOne Terraform Provider version. Provider authentication should be inherited from the root module. Refer to [Provider Authentication Documentation](https://registry.terraform.io/providers/pingidentity/pingone/latest/docs#provider-authentication)

Run `terraform init`

### 3.1 Update Secret Values

PingOne DaVinci has secret attributes (such as client application secrets) that are **not readable via API**. These must be updated manually:

1. Open the generated `ping-export-terraform.auto.tfvars` file
2. Look for and update fields marked as unreadable (`# Secret value - provide manually`)

### 3.2 Run Import Commands

The generated `imports.tf` file contains both commented-out import statements and import blocks:

1. **Import statements**: Run these individually to bring resources under Terraform management
2. **Purpose**: Allows you to update hidden/secret fields in Terraform state before applying the entire configuration

**Example workflow:**

```bash
# Run individual import commands to add resources to state
terraform import module.ping-export.pingone_davinci_connector_instance.pingcli__Variables "b8093f6b-bc03-4c67-af59-eed648c26628/06922a684039827499bdbdd97f49827b"
terraform import module.ping-export.pingone_davinci_connector_instance.pingcli__Flow-0020-Connector "b8093f6b-bc03-4c67-af59-eed648c26628/2581eb287bb1d9bd29ae9886d675f89f"
# ... continue for all resources
```

> Note: If all imports are copy and pasted, they will still run individually and sequentially. With an environment of ~100 resources this takes about 5 minutes.

### 3.3 Update terraform.tfstate Manually

After all resources are imported into Terraform state, you need to manually update obfuscated secret values in the state file.

#### Why This Is Necessary

The DaVinci API doesn't allow reading of attributes that it considers "secrets," such as:

- Connector instance properties like `client_secret`
- Variables with values of type `secret`

When these values are returned by the API, they are obfuscated as `******` (a string of six asterisks). After import, these obfuscated values are stored in your `terraform.tfstate` file and must be replaced with actual secret values.

#### How to Update Secret Values

1. **Locate obfuscated values**: Open the `terraform.tfstate` file and search for the exact string `******`

2. **Replace with actual values**: Update each occurrence with the corresponding actual secret value

   **Example:**

   ```json
    {
      "module": "module.ping-export",
      "mode": "managed",
      "type": "pingone_davinci_connector_instance",
      "name": "pingcli__PingOne-0020-Protect",
      "provider": "provider[\"registry.terraform.io/pingidentity/pingone\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "connector": {
              "id": "pingOneRiskConnector"
            },
            "environment_id": "b8093f6b-bc03-4c67-af59-eed648c26628",
            "id": "292873d5ceea806d81373ed0341b5c88",
            "name": "PingOne Protect",
            "properties": "{\"clientId\":{\"value\":\"b8093f6b-abcd-1234-abcd-eed648c26628\"},\"clientSecret\":{\"value\":\"******\"},\"envId\":{\"value\":\"b8093f6b-abcd-1234-abcd-eed648c26628\"},\"region\":{\"value\":\"NA\"}}"
          },
          "sensitive_attributes": [
            [
              {
                "type": "get_attr",
                "value": "properties"
              }
            ]
          ],
          "identity_schema_version": 0
        }
      ]
    },
   ```

   After updating:

   ```json
    {
      "module": "module.ping-export",
      "mode": "managed",
      "type": "pingone_davinci_connector_instance",
      "name": "pingcli__PingOne-0020-Protect",
      "provider": "provider[\"registry.terraform.io/pingidentity/pingone\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "connector": {
              "id": "pingOneRiskConnector"
            },
            "environment_id": "b8093f6b-bc03-4c67-af59-eed648c26628",
            "id": "292873d5ceea806d81373ed0341b5c88",
            "name": "PingOne Protect",
            "properties": "{\"clientId\":{\"value\":\"b8093f6b-abcd-1234-abcd-eed648c26628\"},\"clientSecret\":{\"value\":\"ACTUAL-VALUE-HERE\"},\"envId\":{\"value\":\"b8093f6b-abcd-1234-abcd-eed648c26628\"},\"region\":{\"value\":\"NA\"}}"
          },
          "sensitive_attributes": [
            [
              {
                "type": "get_attr",
                "value": "properties"
              }
            ]
          ],
          "identity_schema_version": 0
        }
      ]
    },
   ```

3. **Save the state file** once all `******` strings have been replaced

#### Important Notes

- **Only replace the exact string `******`** — this is DaVinci's obfuscation marker
- **Provider behavior**: When resources are **created** by Terraform (not imported), the provider stores the initial value from the declared HCL and only watches for declared changes. If a secret value drifts in the live infrastructure, the drift would not be detected
- **Security reminder**: The `terraform.tfstate` file contains sensitive data. Ensure it's properly secured and never committed to version control

Once all secret values in the state file are updated, you have a complete and accurate representation of your live infrastructure in Terraform state.

## Step 4: Update Configuration References

If your existing Terraform configuration references DaVinci resources (e.g., passing resource IDs to other resources or modules), you need to update these references. Since the export creates a **child module**, you cannot directly reference resources within the module from your root configuration. Instead, you must add outputs to the child module.

### Add Module Outputs

> **Note:** Future enhancements to the tool plan to optionally populate the `outputs.tf` file with common fields automatically, reducing manual configuration.

The export includes an empty `outputs.tf` file in the generated module (e.g., `ping-export/outputs.tf`). Add outputs for any attributes you need to reference from your root module or other configurations.

**Example output definition:**

```hcl
output "dv_app_environment_id" {
  description = "Environment ID for the DaVinci sample application"
  value       = pingone_davinci_application.pingcli__DaVinci-0020-API-0020-Protect-0020-Sample-0020-Application.environment_id
}

output "dv_app_api_key" {
  description = "API key value for the DaVinci sample application"
  value       = pingone_davinci_application.pingcli__DaVinci-0020-API-0020-Protect-0020-Sample-0020-Application.api_key.value
  sensitive   = true
}

output "connector_instance_id" {
  description = "ID of the PingOne Protect connector instance"
  value       = pingone_davinci_connector_instance.pingcli__PingOne-0020-Protect.id
}
```

### Update Root Module References

Once you've added the necessary outputs to the child module, update your root module configuration to reference these outputs:

**Before (legacy provider):**

```hcl
resource "example_resource" "demo" {
  flow_id = davinci_flow.main_auth.id
}
```

**After (using module output):**

```hcl
resource "example_resource" "demo" {
  flow_id = module.ping-export.main_flow_id
}
```

### Finding the Correct Resource Names

To determine the exact resource names for your outputs:

1. Look in the generated module's `.tf` files (e.g., `ping-export/davinci-applications.tf`, `ping-export/davinci-flows.tf`)
2. Find the resource you need to reference
3. Use the full resource reference in your output value

**Tip:** Resource names in the export are prefixed with `pingcli__` and use `-0020-` to represent spaces in the original resource names.

### Validate Configuration

After updating all configuration references, validate your Terraform configuration:

```bash
terraform validate
```

This ensures all module outputs are correctly defined and referenced.

## Step 5: Verify with Terraform Plan

Now run `terraform plan` to verify the migration. Now that you have imported resources with the new provider. updated secret values, and replced dependency reeferences you should see minimal to no functional changes.

```bash
terraform plan -no-color > tfplan-migration.txt 2>&1
```

### What to Expect in the Plan

Because your infrastructure is already managed and imported with the new provider, there should be **no functional changes**. However, you may see:

#### Minor Configuration Updates

- Some resources may receive default values (e.g., flows with a default log level)
- These updates may bump version numbers or trigger deployments
- Any flow that is changed will have a corresponding `pingone_davinci_flow_enable` resource that `will be updated in-place`. This serves as a function to call the API enable endpoint (similar to Save in UI)
- When reading a `pingone_davinci_flow` resource change, attributes with `+` or `-` indicate actual changes. Items prefixed with `~` and ending with `(known after apply)` represent computed attributes to update state—these are non-functional. If a resource plan **only** identifies computed updates, it will not show as an item to change

#### Flow Deploy Resources

- Flow deploy resources will show as "will be created"
- These make API calls to the flow deploy endpoint but don't cause actual changes when the current version equals the deployed version
- Consider these similar to a `terraform import` operation—they bring the deployment status into Terraform state

**Example plan output:**

```text
# module.ping-export.pingone_davinci_flow.my_flow will be updated in-place
  ~ resource "pingone_davinci_flow" "my_flow" {
      ~ current_version   = 1 -> (known after apply)
      ~ settings          = {
          + log_level = 4
        }
        # (4 unchanged attributes hidden)
    }

# module.ping-export.pingone_davinci_flow_deploy.my_flow will be created
+ resource "pingone_davinci_flow_deploy" "my_flow" {
    # No actual change - current version = deployed version
  }
```

## Step 6: Remove Legacy Resources

Finally, remove the legacy DaVinci provider resources from state. Since the new PingOne provider resources are now managing the infrastructure, the legacy resources can be safely removed.

Use the `terraform state rm` command to remove each legacy resource:

```bash
terraform state rm davinci_connection.my_connector
terraform state rm davinci_application.my_app
terraform state rm davinci_flow.my_flow
# ... continue for all legacy resources
```

### Final Verification

After removing all legacy resources:

1. **Run terraform plan**: Should show no changes
   ```bash
   terraform plan
   ```

2. **Apply if needed**: If your plan showed minor configuration updates (default values, deploy resources), run:
   ```bash
   terraform apply
   ```

Your DaVinci resources are now fully migrated to the PingOne Terraform provider!