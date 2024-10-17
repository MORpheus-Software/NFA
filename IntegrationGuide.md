# **Morpheus-Lumerin-Node Integration Guide**

Welcome to the **Morpheus-Lumerin-Node Integration Guide**. This document provides an overview of the smart contract capabilities, API endpoints, and CLI tools available for interacting with the Morpheus Marketplace through the Morpheus-Lumerin-Node project. Whether you're a developer looking to integrate your application or a user aiming to interact with the marketplace directly, this guide will help you get started quickly.

---

## **Table of Contents**

- [Introduction](#introduction)
- [Repository Structure](#repository-structure)
- [Smart Contract Capabilities](#smart-contract-capabilities)
  - [Overview](#overview)
  - [Key Smart Contracts](#key-smart-contracts)
  - [Contract Functions](#contract-functions)
- [API Capabilities](#api-capabilities)
  - [Overview](#overview-1)
  - [Endpoints](#endpoints)
  - [Authentication](#authentication)
  - [Usage Examples](#usage-examples)
- [CLI Capabilities](#cli-capabilities)
  - [Overview](#overview-2)
  - [Installation](#installation)
  - [Commands](#commands)
  - [Usage Examples](#usage-examples-1)
- [Getting Started](#getting-started)
  - [For Developers](#for-developers)
  - [For Users](#for-users)
- [Resources](#resources)
- [Contact & Support](#contact--support)

---

## **Introduction**

The **Morpheus-Lumerin-Node** is an open-source project that provides a node implementation for interacting with the Morpheus Marketplace, a decentralized platform enabling secure and efficient interactions with AI models and services. By leveraging blockchain technology, it ensures transparency, security, and trust in transactions and interactions.

This guide covers three primary layers of interaction with the Morpheus Marketplace via the Morpheus-Lumerin-Node:

1. **Smart Contracts**: Core contracts that define the functionality and rules of the marketplace.
2. **API**: A set of RESTful endpoints that allow applications to interact with the marketplace programmatically.
3. **CLI**: Command-line tools that provide direct interaction with the marketplace for users and developers.

---

## **Repository Structure**

The Morpheus-Lumerin-Node repository is located at [https://github.com/Lumerin-protocol/Morpheus-Lumerin-Node](https://github.com/Lumerin-protocol/Morpheus-Lumerin-Node). The repository structure is organized as follows:

~~~
Morpheus-Lumerin-Node/
├── cmd/
│   └── lumerin/
│       └── main.go
├── internal/
│   ├── blockchainapi/
│   │   ├── controller.go
│   │   ├── service.go
│   │   ├── structs/
│   │   │   └── structs.go
│   │   └── interfaces/
│   │       └── router.go
│   ├── lib/
│   │   ├── logger.go
│   │   └── bigInt.go
│   └── proxyrouter/
│       ├── controller.go
│       ├── service.go
│       └── structs/
│           └── structs.go
├── pkg/
│   └── marketplacesdk/
│       ├── client.go
│       └── models.go
├── go.mod
└── README.md
~~~

- **cmd/lumerin/main.go**: The main entry point of the application.
- **internal/blockchainapi/**: Contains the API controller and service for interacting with the blockchain.
- **internal/proxyrouter/**: Manages proxy routing and session handling.
- **pkg/marketplacesdk/**: The SDK package for interacting with the Morpheus Marketplace API.

---

## **Smart Contract Capabilities**

### **Overview**

The smart contracts are the backbone of the Morpheus Marketplace, handling all decentralized logic and transactions on the blockchain. They manage models, providers, bids, sessions, and transactions involving MOR tokens.

### **Key Smart Contracts**

- **Model Contract**: Manages AI models registered on the marketplace.
- **Provider Contract**: Handles provider registration and staking.
- **Bid Contract**: Facilitates the creation and management of bids by providers.
- **Session Contract**: Manages user sessions with providers and models.
- **Token Contract (MOR)**: The ERC-20 token used within the marketplace for transactions.

### **Contract Functions**

#### **Model Contract**

- **RegisterModel**: Allows users to register a new AI model.
- **DeregisterModel**: Removes a model from the marketplace.
- **GetModel**: Retrieves information about a specific model.
- **ListModels**: Lists all registered models.

#### **Provider Contract**

- **RegisterProvider**: Registers a new provider and stakes MOR tokens.
- **DeregisterProvider**: Removes a provider from the marketplace.
- **UpdateEndpoint**: Updates the provider's service endpoint.
- **GetProvider**: Retrieves provider details.

#### **Bid Contract**

- **CreateBid**: Providers can create bids for their services.
- **DeleteBid**: Removes a bid from the marketplace.
- **GetBid**: Retrieves bid details.
- **ListBidsByModel**: Lists bids associated with a specific model.
- **ListBidsByProvider**: Lists bids from a specific provider.

#### **Session Contract**

- **OpenSession**: Opens a new session between a user and a provider.
- **CloseSession**: Closes an active session.
- **GetSession**: Retrieves session details.
- **ListSessionsByUser**: Lists sessions associated with a user.
- **ListSessionsByProvider**: Lists sessions associated with a provider.

#### **Token Contract (MOR)**

- **Approve**: Approves a spender to transfer MOR tokens on behalf of the user.
- **Transfer**: Transfers MOR tokens to another address.
- **BalanceOf**: Checks the MOR token balance of an address.
- **Allowance**: Checks the allowance set for a spender.

---

## **API Capabilities**

### **Overview**

The Morpheus-Lumerin-Node provides a RESTful API that enables developers to interact with the Morpheus Marketplace programmatically. The API covers functionalities such as managing models, providers, bids, sessions, and transactions.

The API implementation is primarily located in the `internal/blockchainapi/` and `internal/proxyrouter/` directories, particularly in the `controller.go` and `service.go` files.

### **Endpoints**

#### **Models**

- `GET /blockchain/models`: List all models.
- `POST /blockchain/models`: Create a new model.
- `DELETE /blockchain/models/{id}`: Deregister a model.

#### **Providers**

- `GET /blockchain/providers`: List all providers.
- `POST /blockchain/providers`: Register or update a provider.
- `DELETE /blockchain/providers/{id}`: Deregister a provider.

#### **Bids**

- `POST /blockchain/bids`: Create a new bid.
- `GET /blockchain/bids/{id}`: Get bid details.
- `DELETE /blockchain/bids/{id}`: Delete a bid.
- `GET /blockchain/models/{id}/bids`: List bids for a model.
- `GET /blockchain/providers/{id}/bids`: List bids from a provider.

#### **Sessions**

- `POST /blockchain/sessions`: Open a new session.
- `POST /blockchain/sessions/{id}/close`: Close a session.
- `GET /blockchain/sessions/{id}`: Get session details.
- `GET /blockchain/sessions/user`: List sessions for a user.
- `GET /blockchain/sessions/provider`: List sessions for a provider.

#### **Transactions**

- `GET /blockchain/balance`: Get MOR and ETH balances.
- `GET /blockchain/transactions`: List transactions.
- `GET /blockchain/allowance`: Get MOR token allowance.
- `POST /blockchain/approve`: Approve MOR token allowance.
- `POST /blockchain/send/eth`: Send ETH to an address.
- `POST /blockchain/send/mor`: Send MOR tokens to an address.

### **Authentication**

Some API endpoints may require authentication or authorization. Ensure you include any necessary API keys or tokens in your requests. The authentication logic can be customized in the controller or middleware layers.

### **Usage Examples**

#### **Create a New Model**

```http
POST /blockchain/models
Content-Type: application/json

{
  "modelID": "your-model-id",
  "ipfsID": "your-ipfs-hash",
  "fee": "1000000000000000000", // 1 MOR in wei
  "stake": "5000000000000000000", // 5 MOR in wei
  "name": "My AI Model",
  "tags": ["AI", "Machine Learning"]
}

## **Getting Started**

### **For Developers**

1. **Set Up Your Environment**

   - Install Go and any other required dependencies.
   - Clone the **Morpheus-Lumerin-Node** repository.

     ```bash
     git clone https://github.com/Lumerin-protocol/Morpheus-Lumerin-Node.git
     ```

   - Obtain your `PRIVATE_KEY`, `AGENT_ID`, and other necessary credentials.
   - Set up your `.env` file with the required environment variables:

     ```
     API_BASE_URL=http://localhost:8080
     PRIVATE_KEY=your-private-key
     AGENT_ID=your-agent-id
     ```

2. **Build and Run the Node**

   - Navigate to the root directory and build the application:

     ```bash
     cd Morpheus-Lumerin-Node
     go build -o lumerin cmd/lumerin/main.go
     ```

   - Run the application:

     ```bash
     ./lumerin
     ```

3. **Integrate the SDK**

   - Import the `marketplacesdk` package into your Go project:

     ```go
     import marketplacesdk "github.com/Lumerin-protocol/Morpheus-Lumerin-Node/pkg/marketplacesdk"
     ```

   - Initialize the API client with your `API_BASE_URL`:

     ```go
     client := marketplacesdk.NewApiGatewayClient(os.Getenv("API_BASE_URL"), nil)
     ```

   - Use the SDK methods to interact with the marketplace.

4. **Example: Opening a Session**

   ```go
   package main

   import (
       "context"
       "fmt"
       "log"
       "math/big"
       "os"

       marketplacesdk "github.com/Lumerin-protocol/Morpheus-Lumerin-Node/pkg/marketplacesdk"
       "github.com/joho/godotenv"
   )

   func main() {
       // Load .env file
       err := godotenv.Load()
       if err != nil {
           log.Fatal("Error loading .env file")
       }

       // Initialize the Marketplace API client
       apiBaseURL := os.Getenv("API_BASE_URL")
       client := marketplacesdk.NewApiGatewayClient(apiBaseURL, nil)

       // Create session request
       sessionRequest := &marketplacesdk.OpenSessionWithDurationRequest{
           SessionDuration: big.NewInt(3600), // 1 hour
       }

       // Open session
       modelID := os.Getenv("AGENT_ID")
       sessionResponse, err := client.OpenSession(context.Background(), sessionRequest, modelID)
       if err != nil {
           log.Fatalf("Failed to open session: %v", err)
       }

       fmt.Printf("Session opened with ID: %s\n", sessionResponse.SessionID)
   }

   ## **Getting Started**

### **For Developers**

1. **Set Up Your Environment**

   - Install Go and any other required dependencies.
   - Clone the **Morpheus-Lumerin-Node** repository.

     ```bash
     git clone https:github.com/Lumerin-protocol/Morpheus-Lumerin-Node.git
     ```

   - Obtain your `PRIVATE_KEY`, `AGENT_ID`, and other necessary credentials.
   - Set up your `.env` file with the required environment variables:

     ```env
     API_BASE_URL=http:localhost:8080
     PRIVATE_KEY=your-private-key
     AGENT_ID=your-agent-id
     ```

2. **Build and Run the Node**

   - Navigate to the root directory and build the application:

     ```bash
     cd Morpheus-Lumerin-Node
     go build -o lumerin cmd/lumerin/main.go
     ```

   - Run the application:

     ```bash
     ./lumerin
     ```

3. **Integrate the SDK**

   - Import the `marketplacesdk` package into your Go project:

     ```go
     import marketplacesdk "github.com/Lumerin-protocol/Morpheus-Lumerin-Node/pkg/marketplacesdk"
     ```

   - Initialize the API client with your `API_BASE_URL`:

     ```go
     client := marketplacesdk.NewApiGatewayClient(os.Getenv("API_BASE_URL"), nil)
     ```

   - Use the SDK methods to interact with the marketplace.

4. **Example: Opening a Session**

   ```go
   package main

   import (
       "context"
       "fmt"
       "log"
       "math/big"
       "os"

       marketplacesdk "github.com/Lumerin-protocol/Morpheus-Lumerin-Node/pkg/marketplacesdk"
       "github.com/joho/godotenv"
   )

   func main() {
       Load .env file
       err := godotenv.Load()
       if err != nil {
           log.Fatal("Error loading .env file")
       }

       Initialize the Marketplace API client
       apiBaseURL := os.Getenv("API_BASE_URL")
       client := marketplacesdk.NewApiGatewayClient(apiBaseURL, nil)

       Create session request
       sessionRequest := &marketplacesdk.OpenSessionWithDurationRequest{
           SessionDuration: big.NewInt(3600), 1 hour
       }

       Open session
       modelID := os.Getenv("AGENT_ID")
       sessionResponse, err := client.OpenSession(context.Background(), sessionRequest, modelID)
       if err != nil {
           log.Fatalf("Failed to open session: %v", err)
       }

       fmt.Printf("Session opened with ID: %s\n", sessionResponse.SessionID)
   }
   ```

### **For Users**

1. **Install the CLI Tool**

   - Follow the installation instructions in the [CLI Capabilities](#cli-capabilities) section.

2. **Set Up Environment Variables**

   - Create a `.env` file in the same directory as the `lumerin` binary with the following content:

     ```env
     API_BASE_URL=http:localhost:8080
     PRIVATE_KEY=your-private-key
     ```

3. **Check Your Balance**

   ```bash
   ./lumerin balance
   ```

4. **Register as a Provider**

   ```bash
   ./lumerin provider register --endpoint "https:your-endpoint.com" --stake 10
   ```

5. **Interact with Models**

   - List available models:

     ```bash
     ./lumerin model list
     ```

   - Open a session with a model:

     ```bash
     ./lumerin session open --model-id your-model-id --duration 3600
     ```

---

## **Resources**

- **Repository**: [GitHub - Lumerin-protocol/Morpheus-Lumerin-Node](https:github.com/Lumerin-protocol/Morpheus-Lumerin-Node)
- **SDK Package**: [pkg/marketplacesdk](https:github.com/Lumerin-protocol/Morpheus-Lumerin-Node/tree/main/pkg/marketplacesdk)
- **API Documentation**: The API endpoints are defined in the `internal/blockchainapi` and `internal/proxyrouter` directories.
- **Smart Contract Addresses**: Check the `README.md` or the project documentation for deployed contract addresses.

---

## **Contact & Support**

If you have questions or need assistance, please reach out:

- **Email**: support@lumer
