terraform {
    required_version = ">= 1.11.0"

    required_providers {
        azurerm = {
            source  = "hashicorp/azurerm"
            version = ">= 4.0.0"
        }
    }
}

provider "azurerm" {
    features {}
}

resource "azurerm_resource_group" "rg" {
    name     = var.resource_group_name
    location = var.location
}
