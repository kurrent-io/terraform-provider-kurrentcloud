data "kurrentcloud_network" "example" {
  name       = "Example Network"
  project_id = var.project_id
}

output "network_cidr" {
  value = data.kurrentcloud_network.example.cidr_block
}
