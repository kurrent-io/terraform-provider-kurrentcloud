# This assumes a project with the name "Example Project" exists
data "kurrentcloud_project" "example" {
  name = "Example Project"
}

output "project_id" {
  value = data.kurrentcloud_project.example.id
}
