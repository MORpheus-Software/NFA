# UniversalBuilder Nodes for Node-RED

This collection of nodes for Node-RED provides deployment capabilities for Morpheus components to various cloud providers.

## Nodes

### deploy-config
Configuration node that provides shared settings for the deployment nodes.

### deploy-proxy
Deploys the Morpheus Proxy component to the configured cloud provider.

### deploy-consumer
Deploys the Morpheus Consumer component to Google Cloud Run. This node takes the output from the deploy-proxy node and connects the consumer to the proxy.

## Features

- **Google Cloud Run Deployment**: Deploy the Morpheus Consumer to Google Cloud Run with automatic scaling and high availability.
- **Docker Registry Integration**: Pull, tag, and push Docker images to your registry.
- **Secret Management**: Create and manage GCP Secrets for secure credential storage.
- **Health Checking**: Verify deployments are healthy and accessible.
- **Blockchain Configuration**: Configure connections to Ethereum-compatible networks.

## Installation

```bash
cd ~/.node-red
npm install node-red-contrib-universalbuilder
```

Or install using the Node-RED Palette Manager.

## Requirements

- Node-RED v2.0.0 or newer
- Node.js v16 or newer
- Google Cloud SDK installed and configured for Cloud Run deployments
- Docker installed and configured for local image management
- Access to a Docker registry (Docker Hub, GCR, or other)

## Usage

1. Add a `deploy-config` node to your flow to configure shared settings.
2. Connect a `deploy-proxy` node to the config node to deploy the proxy.
3. Connect a `deploy-consumer` node to the proxy node to deploy the consumer.
4. Send a message to the config node to start the deployment flow.

See the examples directory for a complete sample flow.

## Testing

This project includes comprehensive testing for all deployment nodes:

### Unit Tests

Run the unit tests with:

```bash
npm test
```

Unit tests use Sinon to mock external calls to GCP and Docker commands.

### Integration Tests

Integration tests require a GCP account with Cloud Run enabled. Set the `RUN_INTEGRATION_TEST` environment variable to run them:

```bash
# Optional: Configure test settings
export TEST_PROJECT_ID=my-test-project
export TEST_REGION=us-west1
export TEST_DOCKER_REGISTRY=myusername

# Run integration tests
RUN_INTEGRATION_TEST=true node test/integration-test.js
```

**Note**: Integration tests will create and destroy actual Cloud Run services and may incur costs.

## CI/CD

This project uses GitHub Actions for continuous integration:

- Unit tests run on all pull requests and pushes to main
- Integration tests run only on pushes to main when configured

See `.github/workflows/test.yml` for details.

## License

[MIT](LICENSE) 