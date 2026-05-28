# Migration Guide - EventStore Cloud to Kurrent Cloud

This guide provides comprehensive instructions for migrating from the legacy `EventStore/eventstorecloud` provider to the new `kurrent-io/kurrentcloud` provider.

> **📚 For general provider documentation, see the [README.md](README.md) and [Terraform Registry docs](https://registry.terraform.io/providers/kurrent-io/kurrentcloud/latest/docs).**

## Table of Contents

1. [Overview](#overview)
2. [Provider Source Migration](#provider-source-migration)
3. [Terraform State Migration](#terraform-state-migration)
4. [Resource Prefix Migration](#resource-prefix-migration)
5. [Side-by-Side Configuration](#side-by-side-configuration)
6. [Updating Resource Prefixes](#updating-resource-prefixes)
7. [Complete Migration Workflows](#complete-migration-workflows)
8. [Migration Best Practices](#migration-best-practices)
9. [Troubleshooting](#troubleshooting)

## Overview

Starting with version 2.0.0, the Kurrent Cloud Terraform provider has:

- **New provider source**: `kurrent-io/kurrentcloud` (was `EventStore/eventstorecloud`)
- **New resource prefixes**: `kurrentcloud_*` (replaces `eventstorecloud_*`)
- **Backward compatibility**: Both prefixes are supported during the transition period

## Provider Source Migration

### Before (deprecated)
```hcl
terraform {
  required_providers {
    eventstorecloud = {
      source  = "EventStore/eventstorecloud"
      version = "~> 1.0"
    }
  }
}

provider "eventstorecloud" {
  token           = "your-token"
  organization_id = "your-org-id"
}

resource "eventstorecloud_project" "example" {
  name = "my-project"
}
```

### After (recommended)
```hcl
terraform {
  required_providers {
    kurrentcloud = {
      source  = "kurrent-io/kurrentcloud"
      version = "~> 2.0"
    }
  }
}

provider "kurrentcloud" {
  token           = "your-token"
  organization_id = "your-org-id"
}

resource "kurrentcloud_project" "example" {
  name = "my-project"
}
```

## Terraform State Migration

If you have existing resources managed by the old provider, migrate them using the `terraform state replace-provider` command:

### Step 1: Backup Your State
**⚠️ Important:** Always backup your Terraform state before making changes:

```bash
# Create a backup of your current state
terraform state pull > terraform.tfstate.backup
```

### Step 2: Replace Provider in State
Use the `terraform state replace-provider` command to update all resources in your state:

```bash
terraform state replace-provider \
  registry.terraform.io/EventStore/eventstorecloud \
  registry.terraform.io/kurrent-io/kurrentcloud
```

For automatic approval (useful in CI/CD pipelines):
```bash
terraform state replace-provider -auto-approve \
  registry.terraform.io/EventStore/eventstorecloud \
  registry.terraform.io/kurrent-io/kurrentcloud
```

### Step 3: Update Configuration and Reinitialize
After replacing the provider in state, update your Terraform configuration files to use the new provider and run:

```bash
terraform init
```

This will download the new provider and ensure your configuration matches your state.

## Resource Prefix Migration

Starting with version 2.0.0, the provider introduces new resource names with the `kurrentcloud_` prefix. The old `eventstorecloud_` prefixed resources are deprecated but still supported for backward compatibility.

### Available Resources

**New `kurrentcloud_` resources (recommended):**
- `kurrentcloud_project`
- `kurrentcloud_acl`
- `kurrentcloud_network`
- `kurrentcloud_peering`
- `kurrentcloud_managed_cluster`
- `kurrentcloud_scheduled_backup`
- `kurrentcloud_integration`
- `kurrentcloud_integration_awscloudwatch_logs`
- `kurrentcloud_integration_awscloudwatch_metrics`

**Deprecated `eventstorecloud_` resources (still supported):**
- `eventstorecloud_project`
- `eventstorecloud_acl`
- `eventstorecloud_network`
- `eventstorecloud_peering`
- `eventstorecloud_managed_cluster`
- `eventstorecloud_scheduled_backup`
- `eventstorecloud_integration`
- `eventstorecloud_integration_awscloudwatch_logs`
- `eventstorecloud_integration_awscloudwatch_metrics`

### Migration Strategy

You can migrate your resources gradually by updating them one at a time, or continue using the deprecated resource names during a transition period. Both resource types can coexist in the same configuration.

## Side-by-Side Configuration

During the migration period, you can use both the old and new resource prefixes in the same configuration. This allows for gradual migration:

```hcl
terraform {
  required_providers {
    kurrentcloud = {
      source  = "kurrent-io/kurrentcloud"
      version = "= 2.0.0"
    }
    eventstorecloud = {
      source  = "kurrent-io/kurrentcloud"
      version = "= 2.0.0"
    }
  }
}

provider "kurrentcloud" {
  token           = "your-token"
  organization_id = "d2speabtv1luin2fn9dg"
}

provider "eventstorecloud" {
  token           = "your-token"
  organization_id = "d2speabtv1luin2fn9dg"
}

# New resource using kurrentcloud_ prefix
resource "kurrentcloud_project" "new_project" {
  name = "new-project"
}

# Legacy resource using eventstorecloud_ prefix (deprecated)
resource "eventstorecloud_project" "legacy_project" {
  name = "legacy-project"
}
```

**Note:** Both provider aliases point to the same `kurrent-io/kurrentcloud` provider source but allow you to use different resource prefixes during migration.

## Updating Resource Prefixes

After migrating your provider source using `terraform state replace-provider`, you may want to update your resources to use the new `kurrentcloud_` prefix instead of the deprecated `eventstorecloud_` prefix. This section provides detailed examples and methods for safely changing resource prefixes without destroying your existing infrastructure.

### Method 1: Using terraform state mv (All Terraform Versions)

The `terraform state mv` command allows you to rename resources in your Terraform state without destroying the underlying infrastructure.

#### Example: Migrating a Managed Cluster

Let's say you have an existing managed cluster resource that you want to migrate from `eventstorecloud_` to `kurrentcloud_` prefix:

**Original configuration:**
```hcl
resource "eventstorecloud_managed_cluster" "my_cluster" {
  name         = "production-cluster"
  project_id   = "my-project-id"
  network_id   = "my-network-id"
  instance_type = "F1"
  disk_size       = 24
  disk_type       = "GP3"
  disk_iops       = 3000
  disk_throughput = 125
  server_version  = "22.10"
}
```

**Step 1: Backup your state**
```bash
terraform state pull > terraform.tfstate.backup
```

**Step 2: Update your configuration file**
Change the resource type from `eventstorecloud_managed_cluster` to `kurrentcloud_managed_cluster`:

```hcl
resource "kurrentcloud_managed_cluster" "my_cluster" {
  name         = "production-cluster"
  project_id   = "my-project-id"
  network_id   = "my-network-id"
  instance_type = "F1"
  disk_size       = 24
  disk_type       = "GP3"
  disk_iops       = 3000
  disk_throughput = 125
  server_version  = "22.10"
}
```

**Step 3: Move the resource in state**
```bash
terraform state mv eventstorecloud_managed_cluster.my_cluster kurrentcloud_managed_cluster.my_cluster
```

**Step 4: Verify the change**
```bash
terraform plan
```

You should see output like:
```
No changes. Your infrastructure matches the configuration.
```

If you see any destroy/create operations, **DO NOT APPLY** - review your configuration and state mv command.

### Method 2: Using moved Blocks (Terraform v1.1+)

For Terraform v1.1 and later, the modern approach is to use `moved` blocks in your configuration. This method is declarative and documents the resource movement directly in your Terraform code.

#### Example: Migrating a Managed Cluster with moved Block

**Step 1: Add a moved block to your configuration**
Add this temporary configuration to any `.tf` file in your project:

```hcl
# Temporary moved block - can be removed after successful migration
moved {
  from = eventstorecloud_managed_cluster.my_cluster
  to   = kurrentcloud_managed_cluster.my_cluster
}

# Updated resource with new prefix
resource "kurrentcloud_managed_cluster" "my_cluster" {
  name         = "production-cluster"
  project_id   = "my-project-id"
  network_id   = "my-network-id"
  instance_type = "F1"
  disk_size       = 24
  disk_type       = "GP3"
  disk_iops       = 3000
  disk_throughput = 125
  server_version  = "22.10"
}
```

**Step 2: Plan the migration**
```bash
terraform plan
```

You should see output indicating the move operation:
```
Terraform will perform the following actions:

  # eventstorecloud_managed_cluster.my_cluster has moved to kurrentcloud_managed_cluster.my_cluster
    resource "kurrentcloud_managed_cluster" "my_cluster" {
        # (configuration unchanged)
    }

Plan: 0 to add, 0 to change, 0 to destroy.
```

**Step 3: Apply the migration**
```bash
terraform apply
```

**Step 4: Remove the moved block**
After successful application, remove the `moved` block from your configuration. The resource has been permanently moved in your state file.

### Common Resource Migration Examples

Here are quick reference examples for migrating different resource types:

#### Project Migration
```bash
# Using terraform state mv
terraform state mv eventstorecloud_project.my_project kurrentcloud_project.my_project
```

```hcl
# Using moved block
moved {
  from = eventstorecloud_project.my_project
  to   = kurrentcloud_project.my_project
}
```

#### Network Migration
```bash
# Using terraform state mv
terraform state mv eventstorecloud_network.my_network kurrentcloud_network.my_network
```

```hcl
# Using moved block
moved {
  from = eventstorecloud_network.my_network
  to   = kurrentcloud_network.my_network
}
```

#### ACL Migration
```bash
# Using terraform state mv
terraform state mv eventstorecloud_acl.my_acl kurrentcloud_acl.my_acl
```

```hcl
# Using moved block
moved {
  from = eventstorecloud_acl.my_acl
  to   = kurrentcloud_acl.my_acl
}
```

#### Multiple Resources Migration
For multiple resources, you can use multiple `moved` blocks:

```hcl
moved {
  from = eventstorecloud_project.main
  to   = kurrentcloud_project.main
}

moved {
  from = eventstorecloud_network.main
  to   = kurrentcloud_network.main
}

moved {
  from = eventstorecloud_managed_cluster.production
  to   = kurrentcloud_managed_cluster.production
}
```

## Complete Migration Workflows

### Option A: Complete Workflow Using terraform state mv

```bash
# 1. Backup your state
terraform state pull > terraform.tfstate.backup

# 2. List all resources to see what needs migration
terraform state list | grep eventstorecloud_

# 3. For each resource, update configuration files first, then run:
terraform state mv eventstorecloud_project.main kurrentcloud_project.main
terraform state mv eventstorecloud_network.main kurrentcloud_network.main
terraform state mv eventstorecloud_managed_cluster.production kurrentcloud_managed_cluster.production

# 4. Verify no changes needed
terraform plan

# 5. Apply if any configuration drift is detected
terraform apply
```

### Option B: Complete Workflow Using moved Blocks

**Step 1: Create a migration file** (e.g., `migration.tf`):
```hcl
# Temporary migration file - remove after successful migration
moved {
  from = eventstorecloud_project.main
  to   = kurrentcloud_project.main
}

moved {
  from = eventstorecloud_network.main
  to   = kurrentcloud_network.main
}

moved {
  from = eventstorecloud_managed_cluster.production
  to   = kurrentcloud_managed_cluster.production
}
```

**Step 2: Update all resource declarations** in your main configuration files:
```hcl
# Change all eventstorecloud_ prefixes to kurrentcloud_
resource "kurrentcloud_project" "main" {
  # ... configuration
}

resource "kurrentcloud_network" "main" {
  # ... configuration
}

resource "kurrentcloud_managed_cluster" "production" {
  # ... configuration
}
```

**Step 3: Plan and apply the migration:**
```bash
terraform plan  # Should show move operations
terraform apply
```

**Step 4: Clean up migration file:**
```bash
rm migration.tf
```

## Migration Best Practices

### 1. Test in Non-Production First
Always test the migration process in a development or staging environment before applying to production infrastructure.

### 2. Plan Your Migration
- Review all resources that will be affected
- Plan the migration during a maintenance window if possible
- Ensure team members are aware of the migration timeline

### 3. Gradual Migration Approach
- Migrate one resource type at a time
- Use the side-by-side configuration approach for complex setups
- Test each migration step thoroughly

### 4. Version Pinning
Pin your provider version during migration to ensure consistency:
```hcl
terraform {
  required_providers {
    kurrentcloud = {
      source  = "kurrent-io/kurrentcloud"
      version = "= 2.0.0"  # Pin to specific version
    }
  }
}
```

### Migration Safety Checklist

Before performing any resource prefix migration, ensure you:

#### Pre-Migration
- [ ] **Backup your state:** Always run `terraform state pull > terraform.tfstate.backup`
- [ ] **Test in non-production:** Try the migration process in a development environment first
- [ ] **Review dependencies:** Check if any resources reference the ones you're migrating
- [ ] **Team coordination:** Ensure no one else is working on the same Terraform state
- [ ] **Version compatibility:** Ensure you're using a compatible Terraform version

#### During Migration
- [ ] **Verify plan output:** Always run `terraform plan` before `terraform apply`
- [ ] **Check for destroy/create operations:** Should only see "move" operations, never "destroy"
- [ ] **Validate state:** Run `terraform state list` to confirm resources are correctly named
- [ ] **Monitor for errors:** Watch for any error messages during state operations

#### Post-Migration
- [ ] **Run terraform plan:** Should show "No changes" if migration was successful
- [ ] **Test functionality:** Verify your infrastructure still works as expected
- [ ] **Clean up:** Remove temporary migration files (`migration.tf`, etc.)
- [ ] **Document changes:** Update team documentation about the new resource names

## Troubleshooting

### Common Issues and Solutions

#### Issue: `terraform init` fails after state migration
```
Error: Failed to query available provider packages
```
**Solution:** Clear the `.terraform` directory and run `terraform init` again:
```bash
rm -rf .terraform .terraform.lock.hcl
terraform init
```

#### Issue: Provider not found during plan/apply
```
Error: Registry does not have a provider named registry.terraform.io/EventStore/eventstorecloud
```
**Solution:** Ensure you've completed both state migration and configuration updates, then run `terraform init`.

#### Issue: Resource drift detected after migration
**Solution:** This is normal. Run `terraform plan` to review any configuration drift and apply if necessary.

### Resource Prefix Migration Issues

#### Issue: "Resource not found in state"
```
Error: Resource eventstorecloud_managed_cluster.my_cluster not found in state
```
**Solution:** Check the exact resource name in your state:
```bash
terraform state list | grep managed_cluster
```

#### Issue: "Resource already exists"
```
Error: Resource kurrentcloud_managed_cluster.my_cluster already exists in state
```
**Solution:** The resource may have already been moved. Check your current state:
```bash
terraform state show kurrentcloud_managed_cluster.my_cluster
```

#### Issue: Plan shows destroy/create operations
```
# kurrentcloud_managed_cluster.my_cluster will be created
# eventstorecloud_managed_cluster.my_cluster will be destroyed
```
**Solution:**
1. **DO NOT APPLY** - this would destroy your infrastructure
2. Check that you updated your configuration files correctly
3. Verify the state mv command syntax
4. Ensure both resource types point to the same underlying resource schema

#### Issue: Configuration drift after migration
```
Note: Objects have changed outside of Terraform
```
**Solution:** This is normal after migration. Run `terraform plan` to see the changes and `terraform apply` if needed.

### Recovery from Failed Migration
If something goes wrong during migration:

1. **Stop immediately** - don't run additional commands
2. **Restore from backup:**
   ```bash
   terraform state push terraform.tfstate.backup
   ```
3. **Review the issue** and try again with corrected commands
4. **Consider using the alternative method** (moved blocks vs state mv)

### Getting Help

If you encounter issues during migration:
1. Check the [provider documentation](https://registry.terraform.io/providers/kurrent-io/kurrentcloud/latest/docs)
2. Review your backup state file if you need to rollback
3. Open an issue in the [GitHub repository](https://github.com/kurrent-io/terraform-provider-kurrentcloud/issues)