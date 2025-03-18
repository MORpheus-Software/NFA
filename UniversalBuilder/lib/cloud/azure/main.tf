locals {
    merged_web_app_env_vars = merge(var.web_app_env_vars, var.web_app_env_overrides)
    merged_nfa_proxy_env_vars = merge(var.nfa_proxy_env_vars, var.nfa_proxy_env_overrides)
    merged_consumer_node_env_vars = merge(var.consumer_node_env_vars, var.consumer_node_env_overrides)
}

resource "azurerm_container_group" "web_backend" {
    name                = "morhpeus-container-group"
    resource_group_name = azurerm_resource_group.rg.name
    location            = azurerm_resource_group.rg.location
    os_type             = "Linux"

    ##########################
    # Morpheus Consumer Node #
    ##########################

    container {
        name   = "morpheus-consumer-node"
        image  = "${var.consumer_node_image}:${var.consumer_node_image_tag}"
        cpu    = var.consumer_node_cpu
        memory = var.consumer_node_memory

        environment_variables = local.merged_consumer_node_env_vars

        ports {
            port     = var.consumer_node_port
            protocol = "TCP"
        }
    }

    ######################
    # Morpheus NFA Proxy #
    ######################

    container {
        name   = "morpheus-nfa-proxy"
        image  = "${var.nfa_proxy_image}:${var.nfa_proxy_image_tag}"
        cpu    = var.nfa_proxy_cpu
        memory = var.nfa_proxy_memory

        environment_variables = local.merged_nfa_proxy_env_vars

        ports {
            port     = var.nfa_proxy_port
            protocol = "TCP"
        }
    }

    ####################
    # Morpheus Web App #
    ####################

    dynamic "container" {
        for_each = var.deploy_web_app ? [1] : []
        content {
            name   = "morpheus-web-app"
            image  = "${var.web_app_image}:${var.web_app_image_tag}"
            cpu    = var.web_app_cpu
            memory = var.web_app_memory

            environment_variables = local.merged_web_app_env_vars

            ports {
                port     = var.web_app_port
                protocol = "TCP"
            }
        }
    }
}
