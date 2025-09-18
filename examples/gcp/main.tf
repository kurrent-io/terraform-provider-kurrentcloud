terraform {
  required_providers {
    kurrentcloud = {
      source = "kurrent-io/kurrentcloud"
    }
  }
}

variable "peering_route" {
  type = string
}

provider "kurrentcloud" {
}

provider "google" {
}

data "google_project" "project" {
}

data "google_compute_network" "network" {
  name = "default"
}

resource "kurrentcloud_project" "chicken_window" {
  name = "Improved Chicken Window"
}

resource "kurrentcloud_network" "chicken_window" {
  name = "Chicken Window Net"

  project_id = kurrentcloud_project.chicken_window.id

  resource_provider = "gcp"
  region            = "us-central1"
  cidr_block        = "172.29.0.0/16"
}

resource "kurrentcloud_peering" "peering" {
  name = "Example Peering"

  project_id = kurrentcloud_network.chicken_window.project_id
  network_id = kurrentcloud_network.chicken_window.id

  peer_resource_provider = kurrentcloud_network.chicken_window.resource_provider
  peer_network_region    = kurrentcloud_network.chicken_window.region

  peer_account_id = data.google_project.project.project_id
  peer_network_id = data.google_compute_network.network.name
  routes          = [var.peering_route]
}

resource "kurrentcloud_managed_cluster" "wings" {
  name = "Wings Cluster"

  project_id = kurrentcloud_network.chicken_window.project_id
  network_id = kurrentcloud_network.chicken_window.id

  topology         = "three-node-multi-zone"
  instance_type    = "F1"
  disk_size        = 16
  disk_type        = "ssd"
  server_version   = "23.10"
  projection_level = "user"
}

resource "google_compute_network_peering" "example" {
  name                 = "peering"
  network              = data.google_compute_network.default.id
  peer_network         = kurrentcloud_peering.peering.provider_metadata.gcp_network_id
  export_custom_routes = true
  import_custom_routes = true
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
