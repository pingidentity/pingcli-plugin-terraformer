# Managing an Existing PingOne DaVinci Environment with Terraform

This guide walks you through the process of exporting an existing PingOne DaVinci environment, importing its configuration into Terraform state, and establishing a continuous configuration management workflow.

## Overview

This process enables you to:
- Export Terraform HCL configuration from an existing PingOne DaVinci environment
- Import live infrastructure into Terraform state for management
- Establish a continuous configuration management workflow using version control

## Prerequisites

Before you begin, ensure you have:
- An existing PingOne environment with DaVinci configuration
- A worker application with **DaVinci Admin Read Only** role to read the live configuration
- The `pingcli-terraformer` command line tool installed
- Terraform 1.5+ installed
- Git for version control (optional, but recommended)

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

The export will include a versions.tf file that points to the expected PingOne Terraform Provider version. Provider authentication can be added here or use Environment Variables. Refer to [Provider Authentication Documentation](https://registry.terraform.io/providers/pingidentity/pingone/latest/docs#provider-authentication)

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

## Step 4: Review and Apply the Plan

Now you're ready to run `terraform plan` and `terraform apply`.

### Generate a Plan

Output the plan to a file for review:

```bash
terraform plan -no-color > tfplan-1.txt 2>&1
```

### What to Expect in the Plan

Because your infrastructure is already managed and imported, there should be **no functional changes**. However, you may see:

#### Minor Configuration Updates

- Some resources may receive default values (e.g., flows with a default log level)
- These updates may bump version numbers or trigger deployments.
- Any flow that is changed will have a corresponding `pingone_davinci_flow_enable` resource that `will be updated in-place`. This serves as a function to call the API enable endpoint (similar to Save in UI)
- When reading a `pingone_davinci_flow` resource change attributes with `+` or `-` can be considered an actual change. Items leading with a `~` and ending with `(known after apply)` represents a Computed attribute to update state; these are considered non-functional. If a resource plan ONLY identifies Computed updates it will not show as an item to change.

#### Deploy Resources

- Flow deploy resources will show as "will be created".
- These make API calls to the flow deploy endpoint but don't cause actual changes when the current version equals the deployed version
- Consider these similar to a `terraform import` operation — they bring the deployment status into Terraform state

**Example plan output:**

```text
# module.davinci.pingone_davinci_flow.my_flow will be updated in-place
  ~ resource "pingone_davinci_flow" "pingcli__OOTB-0020---0020-Account-0020-Recovery-0020-by-0020-Email" {
    ...
      ~ current_version   = 1 -> (known after apply)
      ~ settings          = {
          + log_level = 4
        }
        # (4 unchanged attributes hidden)
    }

  # module.ping-export.pingone_davinci_flow_enable.pingcli__OOTB-0020---0020-Account-0020-Recovery-0020-by-0020-Email will be updated in-place
  ~ resource "pingone_davinci_flow_enable" "pingcli__OOTB-0020---0020-Account-0020-Recovery-0020-by-0020-Email" {
      ~ enabled        = true -> (known after apply)
        id             = "01af583c6b951086992eb3c37aed7af5"
        # (2 unchanged attributes hidden)
    }
```

### Apply the Configuration

Once you're satisfied with the plan:

```bash
terraform apply
```

Now this DaVinci environment can be considered completely managed by Terraform.

## Step 5: Establish Continuous Development

With your existing environment now managed by Terraform, you can establish a continuous configuration management workflow.

### Development Lifecycle

The typical development lifecycle includes:

1. **Build features or changes** in the PingOne UI
2. **Export those changes** using `pingcli-terraformer export`
3. **View changes with git** using IDE, `git diff --unified=0` or similar
4. **Validate changes in Terraform** `terraform plan` should refresh state against live environment and result in no changes needed
5. **Commit to version control** so changes can be picked up by automated pipelines
6. **Promote to higher environments** through your CI/CD pipeline

### Initialize Version Control

If you haven't already, initialize a Git repository:

```bash
git init
```

Create a `.gitignore` file to exclude sensitive and generated files:

```gitignore
# Terraform
.terraform/
.terraform.lock.hcl
*.tfstate
*.tfstate.*
*.tfvars
crash.log
override.tf
override.tf.json
*_override.tf
*_override.tf.json
tfplan*

# Sensitive or environment specific
*.pem
*.key
.env
.env.*
*.tfvars
*imports.tf
```

### Commit Your Configuration

```bash
git add .
git commit -m "Initial Terraform configuration for DaVinci environment"
```

### Next Steps

- Set up a CI/CD pipeline to automate terraform apply for higher environments
- Establish branching strategies for development, staging, and production
- Document your specific workflow and approval processes
- Consider using Terraform workspaces or separate state files per environment

## Troubleshooting

### Common Issues

**Import failures:**
- Verify your terraform provider worker application has the correct DaVinci Admin R role
- Check that resource IDs in the imports file are correct
- Ensure your Terraform version is 1.5 or higher

**Secret value errors:**
- Double-check that all UNREADABLE fields in `terraform.tfvars` have been updated
- Verify secrets are correctly formatted (no extra spaces or newlines)

**Plan shows unexpected changes:**
- Review the diff carefully—some default values may be applied
- Check if API responses have changed since export
- Re-run export if the environment was modified during the import process

## Additional Resources

- [Ping CLI Terraformer Repository](https://github.com/pingidentity/pingcli-plugin-terraformer)
- [Terraform Import Documentation](https://www.terraform.io/docs/cli/import/index.html)
- [PingOne DaVinci Documentation](https://docs.pingidentity.com/davinci)

## Summary

You've successfully:
✓ Exported an existing DaVinci environment to Terraform HCL  
✓ Imported the live infrastructure into Terraform state  
✓ Applied the configuration with Terraform  
✓ Established a foundation for continuous configuration management  

Your PingOne DaVinci environment is now managed by Terraform and ready for version-controlled, automated deployments.
