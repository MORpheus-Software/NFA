#################
# Azure Configs #
#################

variable "resource_group_name" {
    type = string
    default = "morpheus"
}

variable "location" {
    type = string
    default = "East US"
    description = "Azure region the Morpheus stack will be deployed in"
}

variable "environment" {
    type = string
}

#########################
# Consumer Node Configs #
#########################

variable "consumer_node_image" {
    type = string
    default = "docker.io/srt0422/morpheus-marketplace-consumer"
}

variable "consumer_node_image_tag" {
    type = string
    default = "latest"
}

variable "consumer_node_port" {
    type = number
    default = 8082
    description = "Service port to expose"
}

variable "consumer_node_cpu" {
    type = number
    default = 0.5
    description = "The number of vCPU cores available to the container"
}

variable "consumer_node_memory" {
    type = number
    default = 1.5
    description = "The amount of RAM (in GB) available to the container"
}

variable "consumer_node_env_vars" {
  type = map(string)
  default = {
    BLOCKCHAIN_HTTP_URL="" # "wss://arbitrum-mainnet.infura.io/ws/v3/your-project-id"
    BLOCKCHAIN_WS_URL="" # "https://arbitrum-mainnet.infura.io/v3/your-project-id" public endpoint - https://sepolia-rollup.arbitrum.io/rpc
    BLOCKSCOUT_API_URL="https://api-sepolia.arbiscan.io/api"
    COOKIE_FILE_PATH="/secrets/.cookie"
    CONSUMER_PASSWORD="your-secure-password"
    CONSUMER_USERNAME="admin"
    DIAMOND_CONTRACT_ADDRESS="0xb8C55cD613af947E73E262F0d3C54b7211Af16CF"
    ENVIRONMENT="production"
    ETH_NODE_ADDRESS="https://sepolia-rollup.arbitrum.io/rpc"
    ETH_NODE_CHAIN_ID="421614"
    ETH_NODE_LEGACY_TX=false
    ETH_NODE_USE_SUBSCRIPTIONS=false
    EXPLORER_API_URL="https://api-sepolia.arbiscan.io/api"
    GO_ENV="production"
    LOG_COLOR=true
    LOG_FORMAT="text"
    LOG_LEVEL="info"
    MAX_CONCURRENT_SESSIONS=100
    MOR_TOKEN_ADDRESS="0x34a285a1b1c166420df5b6630132542923b5b27e"
    PROXY_ADDRESS="0.0.0.0:3333"
    PROXY_STORE_CHAT_CONTEXT=true
    PROXY_STORAGE_PATH="./data/"
    PROVIDER_CACHE_TTL=60
    SESSION_TIMEOUT=3600
    WALLET_PRIVATE_KEY="your-wallet-key"
    WEB_ADDRESS="0.0.0.0:8082"
  }
}

variable "consumer_node_env_overrides" {
  type    = map(string)
  default = {}
  description = "Used to override a particular consumer_node_env_vars value"
}

#####################
# NFA Proxy Configs #
#####################

variable "nfa_proxy_image" {
    type = string
    default = "docker.io/srt0422/openai-morpheus-proxy"
}

variable "nfa_proxy_image_tag" {
    type = string
    default = "latest"
}

variable "nfa_proxy_port" {
    type = number
    default = 8082
    description = "Service port to expose"
}

variable "nfa_proxy_cpu" {
    type = number
    default = 0.5
    description = "The number of vCPU cores available to the container"
}

variable "nfa_proxy_memory" {
    type = number
    default = 1.5
    description = "The amount of RAM (in GB) available to the container"
}

variable "nfa_proxy_env_vars" {
  type = map(string)
  default = {
    CONSUMER_USERNAME="admin"
    CONSUMER_PASSWORD="your-secure-password"
    CONSUMER_NODE_URL="http://morpheus-consumer-node:8082"
    INTERNAL_API_PORT=8082
    MARKETPLACE_PORT=3333
    MARKETPLACE_BASE_URL="http://morpheus-consumer-node:8082"
    MARKETPLACE_URL="http://morpheus-consumer-node:8082"
    SESSION_DURATION= "1h"
  }
}

variable "nfa_proxy_env_overrides" {
  type    = map(string)
  default = {}
  description = "Used to override a particular nfa_proxy_env_vars value"
}

###################
# Web App Configs #
###################

variable "deploy_web_app" {
    type = bool
    default = true
    description = "Enables deployment of the frontend Chat application"
}

variable "web_app_image" {
    type = string
    default = "docker.io/srt0422/chat-web-app"
}

variable "web_app_image_tag" {
    type = string
    default = "latest"
}

variable "web_app_port" {
    type = number
    default = 8080
    description = "Service port to expose"
}

variable "web_app_cpu" {
    type = number
    default = 0.5
    description = "The number of vCPU cores available to the container"
}

variable "web_app_memory" {
    type = number
    default = 1.5
    description = "The amount of RAM (in GB) available to the container"
}

variable "web_app_env_vars" {
  type = map(string)
  default = {
    CHAT_COMPLETIONS_PATH="/v1/chat/completions"
    MODEL_NAME="LMR-Hermes-2-Theta-Llama-3-8B"
    NEXT_PUBLIC_CHAT_COMPLETIONS_PATH="/v1/chat/completions"
    OPENAI_API_URL="http://morpheus-nfa-proxy:8082"
  }
}

variable "web_app_env_overrides" {
  type    = map(string)
  default = {}
  description = "Used to override a particular web_app_env_vars value"
}