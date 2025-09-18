resource "kurrentcloud_acl" "example" {
  name       = "Example IP Access List"
  project_id = "example-project-id"
  cidr_blocks = [
    {
      cidr        = "192.168.1.0/24"
      description = "Office network"
    },
    {
      cidr        = "10.0.0.0/16"
      description = "VPN network"
    }
  ]
}