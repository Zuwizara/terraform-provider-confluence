provider "confluence" {
  cloud_id  = var.cloud_id
  api_token = var.api_token
}

resource "confluence_attachment" "example" {
  title   = "example.txt"
  data    = "This is the contents of the example attachment."
  page_id = confluence_page.example.id
}

data "confluence_space" "example" {
  key = var.space
}

resource "confluence_page" "example" {
  title    = "Example Page"
  body     = "This page has a <ac:link><ri:attachment ri:filename=\"example.txt\"/><ac:plain-text-link-body><![CDATA[file attachment]]></ac:plain-text-link-body></ac:link>."
  space_id = data.confluence_space.example.id
}

terraform {
  required_version = ">= 1.0"
  required_providers {
    confluence = {
      source = "zuwizara/confluence"
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

output "example_content" {
  value = confluence_page.example
}

output "example_attachment" {
  value = confluence_attachment.example
}
