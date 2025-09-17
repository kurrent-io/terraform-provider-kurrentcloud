---
subcategory: ""
page_title: "Provision Kurrent Cloud resources in AWS"
description: |-
    A sample Terraform project to provision all the Kurrent Cloud resources in AWS.
---

# Kurrent Cloud in AWS

The sample project creates the following resources in Kurrent Cloud:
- Project
- Network
- Network peering
- Managed KurrentDB using single F1 node with 16GB disk

From the AWS side, you still need to accept the peering request and configure the route as described in the [documentation](https://developers.eventstore.com/cloud/provision/aws/#network-peering).
This step can be also automated using the AWS Terraform provider.

```terraform
terraform {
  required_providers {
    kurrentcloud = {
      source = "kurrent-io/kurrentcloud"
    }
  }
}

provider "aws" {

}

provider "kurrentcloud" {
}

data "aws_caller_identity" "example" {
}

resource "aws_vpc" "example" {
  cidr_block = "172.250.0.0/24"

  tags = {
    Name = "eventstore-example"
  }
}

resource "kurrentcloud_project" "chicken_window" {
  name = "Improved Chicken Window"
}

resource "kurrentcloud_network" "chicken_window" {
  name = "Chicken Window Net"

  project_id = kurrentcloud_project.chicken_window.id

  resource_provider = "aws"
  region            = "us-west-2"
  cidr_block        = "172.21.0.0/16"
}

resource "kurrentcloud_peering" "peering" {
  name = "Example Peering"

  project_id = kurrentcloud_network.chicken_window.project_id
  network_id = kurrentcloud_network.chicken_window.id

  peer_resource_provider = kurrentcloud_network.chicken_window.resource_provider
  peer_network_region    = kurrentcloud_network.chicken_window.region

  peer_account_id = data.aws_caller_identity.example.account_id
  peer_network_id = aws_vpc.example.id
  routes          = [aws_vpc.example.cidr_block]
}

resource "aws_vpc_peering_connection_accepter" "peer" {
  vpc_peering_connection_id = kurrentcloud_peering.peering.provider_metadata.aws_peering_link_id
  auto_accept               = true

  tags = {
    Side   = "Accepter"
    Source = "Event Store"
  }
}

resource "aws_route" "peering" {
  route_table_id            = aws_vpc.example.main_route_table_id
  destination_cidr_block    = kurrentcloud_network.chicken_window.cidr_block
  vpc_peering_connection_id = aws_vpc_peering_connection_accepter.peer.id
}

resource "kurrentcloud_managed_cluster" "wings" {
  name = "Wings Cluster"

  project_id = kurrentcloud_network.chicken_window.project_id
  network_id = kurrentcloud_network.chicken_window.id

  topology        = "single-node"
  instance_type   = "F1"
  disk_size       = 16
  disk_type       = "gp3"
  disk_iops       = 3000
  disk_throughput = 125
  server_version  = "23.10"
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
