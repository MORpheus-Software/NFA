const { spawn } = require('child_process');
const helper = require('node-red-node-test-helper');
const deployConsumer = require('../nodes/deploy-consumer.js');
const deployProxy = require('../nodes/deploy-proxy.js');
const deployConfig = require('../nodes/deploy-config.js');
const axios = require('axios');

// Only run this if explicitly asked for an integration test
if (process.env.RUN_INTEGRATION_TEST !== 'true') {
    console.log('Skipping integration test. Set RUN_INTEGRATION_TEST=true to enable.');
    process.exit(0);
}

// You can override these with environment variables
const testConfig = {
    projectId: process.env.TEST_PROJECT_ID || 'morpheus-test-project',
    region: process.env.TEST_REGION || 'us-west1',
    dockerRegistry: process.env.TEST_DOCKER_REGISTRY || 'srt0422',
    proxyVersion: process.env.TEST_PROXY_VERSION || 'v0.0.31',
    consumerVersion: process.env.TEST_CONSUMER_VERSION || 'v0.0.19',
    consumerUsername: process.env.TEST_USERNAME || 'test-admin',
    consumerPassword: process.env.TEST_PASSWORD || 'test-password'
};

async function runIntegrationTest() {
    console.log('Starting integration test with config:', testConfig);
    
    // Initialize Node-RED
    helper.init(require.resolve('node-red'));
    
    try {
        await helper.startServer();
        
        // Create the test flow with all required nodes
        const flow = [
            {
                id: "config",
                type: "deploy-config",
                name: "Test Config",
                projectId: testConfig.projectId,
                region: testConfig.region,
                dockerRegistry: testConfig.dockerRegistry,
                wires: [["proxy"]]
            },
            {
                id: "proxy",
                type: "deploy-proxy",
                name: "Test Proxy",
                projectId: testConfig.projectId,
                region: testConfig.region,
                dockerRegistry: testConfig.dockerRegistry,
                proxyVersion: testConfig.proxyVersion,
                internalApiPort: "8080",
                marketplacePort: "3333",
                sessionDuration: "1h",
                consumerUsername: testConfig.consumerUsername,
                consumerPassword: testConfig.consumerPassword,
                wires: [["consumer"]]
            },
            {
                id: "consumer",
                type: "deploy-consumer",
                name: "Test Consumer",
                projectId: testConfig.projectId,
                region: testConfig.region,
                dockerRegistry: testConfig.dockerRegistry,
                consumerVersion: testConfig.consumerVersion,
                consumerUsername: testConfig.consumerUsername,
                consumerPassword: testConfig.consumerPassword,
                walletKey: process.env.TEST_WALLET_KEY || "",
                contractAddress: "0xb8C55cD613af947E73E262F0d3C54b7211Af16CF",
                morTokenAddress: "0x34a285a1b1c166420df5b6630132542923b5b27e",
                blockchainHttpUrl: "https://sepolia-rollup.arbitrum.io/rpc",
                explorerApiUrl: "https://api-sepolia.arbiscan.io/api",
                ethNodeChainId: "421614",
                wires: [["output"]]
            },
            { id: "output", type: "helper" }
        ];
        
        // Load the nodes
        await helper.load([deployConfig, deployProxy, deployConsumer], flow);
        
        // Get the nodes
        const configNode = helper.getNode("config");
        const outputNode = helper.getNode("output");
        
        // Services to clean up
        let deployedServices = {
            proxy: null,
            consumer: null
        };
        
        // Wait for the complete flow to execute
        return new Promise((resolve, reject) => {
            const timeout = setTimeout(() => {
                reject(new Error('Integration test timed out after 15 minutes'));
            }, 15 * 60 * 1000);
            
            outputNode.on("input", async (msg) => {
                try {
                    console.log('Consumer deployment result:', msg.payload);
                    
                    // Store service URLs for cleanup
                    if (msg.config && msg.config.consumerUrl) {
                        deployedServices.consumer = getServiceName(msg.config.consumerUrl);
                    }
                    if (msg.config && msg.config.proxyUrl) {
                        deployedServices.proxy = getServiceName(msg.config.proxyUrl);
                    }
                    
                    // Verify we have a consumer URL
                    const consumerUrl = msg.payload.consumerUrl;
                    if (!consumerUrl) {
                        throw new Error('No consumer URL in deployment result');
                    }
                    
                    // Check the consumer health
                    console.log('Checking consumer health at:', consumerUrl);
                    try {
                        const response = await axios.get(`${consumerUrl}/healthcheck`, {
                            timeout: 30000 // 30 seconds timeout
                        });
                        
                        if (response.status !== 200) {
                            throw new Error(`Health check failed with status: ${response.status}`);
                        }
                        
                        console.log('Consumer health check passed:', response.data);
                    } catch (err) {
                        console.error('Health check request failed:', err.message);
                        throw new Error(`Health check request failed: ${err.message}`);
                    }
                    
                    // Success!
                    console.log('Integration test passed! 🎉');
                    clearTimeout(timeout);
                    resolve(deployedServices);
                } catch (err) {
                    console.error('Test execution error:', err);
                    clearTimeout(timeout);
                    reject(err);
                }
            });
            
            // Start the flow by sending a message to the config node
            console.log('Triggering deployment flow...');
            configNode.receive({ payload: {} });
        });
    } finally {
        await helper.unload();
        await helper.stopServer();
    }
}

function getServiceName(url) {
    if (!url) return null;
    
    try {
        const urlObj = new URL(url);
        return urlObj.hostname.split('.')[0]; // Extract service name from host
    } catch (err) {
        console.error('Failed to parse URL:', url, err);
        return null;
    }
}

async function cleanupResources(services) {
    console.log('Cleaning up deployed resources:', services);
    
    for (const [type, serviceName] of Object.entries(services)) {
        if (serviceName) {
            try {
                console.log(`Deleting ${type} service: ${serviceName}`);
                
                const cleanup = spawn('gcloud', [
                    'run', 'services', 'delete', serviceName,
                    '--region', testConfig.region,
                    '--project', testConfig.projectId,
                    '--quiet'
                ]);
                
                let output = '';
                
                cleanup.stdout.on('data', (data) => {
                    output += data.toString();
                    console.log(`${type} cleanup: ${data}`);
                });
                
                cleanup.stderr.on('data', (data) => {
                    output += data.toString();
                    console.error(`${type} cleanup error: ${data}`);
                });
                
                await new Promise((resolve, reject) => {
                    cleanup.on('close', (code) => {
                        console.log(`${type} cleanup process exited with code ${code}`);
                        if (code === 0) {
                            resolve();
                        } else {
                            console.error(`Failed to delete ${type} service: ${output}`);
                            resolve(); // Continue even if cleanup fails
                        }
                    });
                });
            } catch (error) {
                console.error(`Error during ${type} cleanup:`, error);
            }
        }
    }
}

// Run the integration test
runIntegrationTest()
    .then((services) => {
        console.log('Integration test completed successfully');
        return cleanupResources(services);
    })
    .then(() => {
        console.log('Cleanup completed');
        process.exit(0);
    })
    .catch(async (error) => {
        console.error('Integration test failed:', error);
        process.exit(1);
    }); 