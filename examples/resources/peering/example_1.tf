# Example for AWS

resource "kurrentcloud_project" "example" {
  name = "Example Project"
}

resource "kurrentcloud_network" "example" {
  name = "Example Network"

  project_id = kurrentcloud_project.example.id

  resource_provider = "aws"
  region            = "us-west-2"
  cidr_block        = "172.21.0.0/16"
}

resource "kurrentcloud_peering" "example" {
  name = "Peering from AWS into Example Network"

  project_id = kurrentcloud_network.example.project_id
  network_id = kurrentcloud_network.example.id

  peer_resource_provider = kurrentcloud_network.example.resource_provider
  peer_network_region    = kurrentcloud_network.example.region

  peer_account_id = "<Customer AWS Account ID>"
  peer_network_id = "<Customer VPC ID>"
  routes          = ["<Address space of the customer VPC>"]
}
