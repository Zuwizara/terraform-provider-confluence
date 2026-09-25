# Example: keep track of your pets in Confluence

resource "random_pet" "pets" {
  count     = 4
  separator = " "
  length    = 3
}

provider "confluence" {
  cloud_id  = var.cloud_id
  api_token = var.api_token
}

data "confluence_space" "example" {
  key = var.space
}

resource "confluence_page" "example" {
  space_id = data.confluence_space.example.id
  title    = "My Pets"
  body = templatefile("${path.module}/example.tmpl", {
    pets = [for p in random_pet.pets : title(p.id)]
  })
}

terraform {
  required_version = ">= 1.0"
  required_providers {
    confluence = {
      source = "zuwizara/confluence"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 2.2"
    }
  }
}

variable "cloud_id" {
  type = string
}

variable "api_token" {
  type      = string
  sensitive = true
}

variable "space" {
  type = string
}

output "example" {
  value = confluence_page.example
}
