---
subcategory: ""
page_title: "Provision Kurrent Cloud resources in Azure"
description: |-
    A sample Terraform project to provision all the Kurrent Cloud resources in Azure.
---

# Kurrent Cloud in Azure

The sample project creates the following resources in Kurrent Cloud:
- Project
- Network
- Network peering
- Managed KurrentDB using single F1 node with 16GB disk

From the Azure side, you still need to register the Kurrent Cloud service account with the necessary permissions to create a network peering.
Read more in the provisioning [documentation](https://developers.eventstore.com/cloud/provision/azure/#network-peering).

```terraform
terraform {
  required_providers {
    kurrentcloud = {
      source = "kurrent-io/kurrentcloud"
    }
    azurerm = {
      source = "hashicorp/azurerm"
    }
    azuread = {
      source = "hashicorp/azuread"
    }
  }
}

variable "eventstore_application_id" {
  type        = string
  description = "Event Store Production Application ID with access to peering creation"
  default     = "38bd60cb-6efa-49e8-a1cd-3b9f61d9435e"
}

variable "azure_subscription_id" {
  type        = string
  description = "Azure Subscruption ID"
}

variable "esc_token" {
  type        = string
  description = "Event Store Cloud API token"
}

variable "esc_organization_id" {
  type        = string
  description = "Event Store Cloud Organization ID"
}

provider "azurerm" {
  features {}
  subscription_id            = var.azure_subscription_id
  skip_provider_registration = false
}

provider "azuread" {}

provider "kurrentcloud" {
  token           = var.esc_token
  organization_id = var.esc_organization_id
}

data "azurerm_client_config" "current" {}

data "azuread_client_config" "current" {}

data "azurerm_subscription" "current" {}

resource "azurerm_resource_group" "chicken_window" {
  name     = "chicken-window"
  location = "West US2"
}

resource "kurrentcloud_project" "chicken_window" {
  name = "Improved Chicken Window"
}

resource "azurerm_virtual_network" "chicken_window" {
  name                = "chicken-window-network"
  resource_group_name = azurerm_resource_group.chicken_window.name
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.chicken_window.location
}

resource "kurrentcloud_network" "chicken_window" {
  name = "Chicken Window Net"

  project_id = kurrentcloud_project.chicken_window.id

  resource_provider = "azure"
  region            = azurerm_resource_group.chicken_window.location
  cidr_block        = "10.2.0.0/16"
}

// Access to Event Store Application should be granted to create peering between Azure Virtual Networks
resource "azuread_service_principal" "peering" {
  client_id                    = var.eventstore_application_id
  app_role_assignment_required = false
}

resource "azurerm_role_definition" "chicken_window_peering" {
  name        = "ESCPeering/${data.azurerm_subscription.current.id}/${azurerm_resource_group.chicken_window.name}/${azurerm_virtual_network.chicken_window.name}"
  scope       = data.azurerm_subscription.current.id
  description = "Grants ESC access to manage peering connections on network ${azurerm_virtual_network.chicken_window.id}"

  permissions {
    actions = [
      "Microsoft.Network/virtualNetworks/virtualNetworkPeerings/read",
      "Microsoft.Network/virtualNetworks/virtualNetworkPeerings/write",
      "Microsoft.Network/virtualNetworks/virtualNetworkPeerings/delete",
      "Microsoft.Network/virtualNetworks/peer/action"
    ]
    not_actions = []
  }

  assignable_scopes = [
    azurerm_virtual_network.chicken_window.id,
  ]
}

resource "azurerm_role_assignment" "chicken_window_peering" {
  scope                = azurerm_virtual_network.chicken_window.id
  role_definition_name = azurerm_role_definition.chicken_window_peering.name
  principal_id         = azuread_service_principal.peering.id
}

resource "kurrentcloud_peering" "peering" {
  name = "Example Peering"

  project_id = kurrentcloud_network.chicken_window.project_id
  network_id = kurrentcloud_network.chicken_window.id

  peer_resource_provider = kurrentcloud_network.chicken_window.resource_provider
  peer_network_region    = kurrentcloud_network.chicken_window.region

  peer_account_id = data.azurerm_client_config.current.tenant_id
  peer_network_id = azurerm_virtual_network.chicken_window.id
  routes          = azurerm_virtual_network.chicken_window.address_space

  depends_on = [
    azurerm_role_assignment.chicken_window_peering,
  ]
}

resource "kurrentcloud_managed_cluster" "wings" {
  name = "Wings Cluster"

  project_id = kurrentcloud_network.chicken_window.project_id
  network_id = kurrentcloud_network.chicken_window.id

  topology         = "three-node-multi-zone"
  instance_type    = "F1"
  disk_size        = 10
  disk_type        = "premium-ssd-lrs"
  server_version   = "23.10"
  projection_level = "user"
}

output "chicken_window_id" {
  value = kurrentcloud_project.chicken_window.id
}

output "chicken_window_net" {
  value = kurrentcloud_network.chicken_window
}

output "chicken_window_peering" {
  value = kurrentcloud_peering.peering
}

output "wings_cluster_dns_name" {
  value = kurrentcloud_managed_cluster.wings.dns_name
}
```
