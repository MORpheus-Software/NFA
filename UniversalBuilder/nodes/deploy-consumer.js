const { exec } = require('child_process');
const util = require('util');
const fs = require('fs');
const path = require('path');
const execAsync = util.promisify(exec);
const fsPromises = fs.promises;

module.exports = function(RED) {
    function ConsumerNode(config) {
        RED.nodes.createNode(this, config);
        const node = this;

        // Add label function
        this.name = config.name;

        // Store GCP deployment configuration
        node.projectId = config.projectId;
        node.region = config.region;
        node.dockerRegistry = config.dockerRegistry;
        node.consumerVersion = config.consumerVersion;
        node.consumerUsername = config.consumerUsername;
        node.consumerPassword = config.consumerPassword;
        node.useCookieSecret = config.useCookieSecret;
        node.cookieSecretName = config.cookieSecretName || "COOKIE_SECRET";

        // Store blockchain configuration values
        node.walletKey = config.walletKey;
        node.contractAddress = config.contractAddress;
        node.morTokenAddress = config.morTokenAddress;
        node.blockchainWsUrl = config.blockchainWsUrl;
        node.blockchainHttpUrl = config.blockchainHttpUrl;
        node.explorerApiUrl = config.explorerApiUrl;
        node.ethNodeChainId = config.ethNodeChainId;
        node.ethNodeLegacyTx = config.ethNodeLegacyTx;
        node.ethNodeUseSubscriptions = config.ethNodeUseSubscriptions;

        // Store service configuration values
        node.proxyAddress = config.proxyAddress;
        node.webAddress = config.webAddress;
        node.webPublicUrl = config.webPublicUrl;
        node.environment = config.environment;
        node.proxyStoreChatContext = config.proxyStoreChatContext;
        node.proxyStoragePath = config.proxyStoragePath;
        node.logLevel = config.logLevel;
        node.logFormat = config.logFormat;
        node.logColor = config.logColor;
        node.providerCacheTtl = config.providerCacheTtl;
        node.maxConcurrentSessions = config.maxConcurrentSessions;
        node.sessionTimeout = config.sessionTimeout;

        // Use execAsync function from this instance for testing
        this._execAsync = execAsync;

        node.on('input', async function(msg) {
            try {
                node.status({fill:"blue", shape:"dot", text:"Creating secret..."});
                await createOrUpdateCookieSecret(node);
                
                node.status({fill:"blue", shape:"dot", text:"Preparing image..."});
                await prepareDockerImage(node, msg);
                
                node.status({fill:"blue", shape:"dot", text:"Deploying consumer..."});
                const deployResult = await deployConsumerToCloudRun(node, msg);
                
                node.status({fill:"blue", shape:"dot", text:"Checking health..."});
                await checkConsumerHealth(node, deployResult.consumerUrl);
                
                // Update msg with deployment results
                msg.config = {
                    ...msg.config,
                    consumerUrl: deployResult.consumerUrl
                };
                
                msg.payload = {
                    status: 'success',
                    action: 'deploy',
                    consumerUrl: deployResult.consumerUrl,
                    output: deployResult.output
                };
                
                node.status({fill:"green", shape:"dot", text:"Deployed: " + deployResult.consumerUrl});
                node.send(msg);
            } catch (error) {
                handleDeploymentError(node, msg, error);
            }
        });
    }

    async function createOrUpdateCookieSecret(node) {
        try {
            // Ensure that both consumerUsername and consumerPassword are set
            if (!node.consumerUsername || !node.consumerPassword) {
                throw new Error("Both consumerUsername and consumerPassword must be set.");
            }

            // Generate the .cookie file content
            const cookieContent = `${node.consumerUsername}:${node.consumerPassword}`;
            
            // Create a temporary file with the cookie content
            const tmpPath = path.join(RED.settings.userDir, '.tmp');
            
            // Ensure the temp directory exists
            try {
                await fsPromises.mkdir(tmpPath, { recursive: true });
            } catch (err) {
                if (err.code !== 'EEXIST') throw err;
            }
            
            const tmpCookieFile = path.join(tmpPath, `cookie_${Date.now()}.txt`);
            await fsPromises.writeFile(tmpCookieFile, cookieContent);

            // Check if the secret exists
            const checkSecretCmd = `gcloud secrets describe ${node.cookieSecretName} 2>/dev/null || echo "Secret does not exist"`;
            const { stdout: checkResult } = await node._execAsync(checkSecretCmd);

            let secretCmd;
            if (checkResult.includes("Secret does not exist")) {
                // Create the secret
                secretCmd = `gcloud secrets create ${node.cookieSecretName} --data-file="${tmpCookieFile}" --replication-policy="automatic"`;
            } else {
                // Add new version to existing secret
                secretCmd = `gcloud secrets versions add ${node.cookieSecretName} --data-file="${tmpCookieFile}"`;
            }

            await node._execAsync(secretCmd);
            
            // Clean up temporary file
            await fsPromises.unlink(tmpCookieFile);
            
            return true;
        } catch (error) {
            throw new Error(`Failed to create or update cookie secret: ${error.message}`);
        }
    }

    async function prepareDockerImage(node, msg) {
        try {
            const imageTag = node.consumerVersion;
            const sourceImage = `${node.dockerRegistry}/morpheus-marketplace-consumer:${imageTag}`;
            const targetImage = `gcr.io/${node.projectId}/morpheus-lumerin-node:${imageTag}`;
            
            // Pull the image from Docker Hub
            const pullCmd = `docker pull ${sourceImage}`;
            await node._execAsync(pullCmd);
            
            // Tag the image for Google Container Registry
            const tagCmd = `docker tag ${sourceImage} ${targetImage}`;
            await node._execAsync(tagCmd);
            
            // Push the image to Google Container Registry
            const pushCmd = `docker push ${targetImage}`;
            await node._execAsync(pushCmd);
            
            return {
                sourceImage,
                targetImage
            };
        } catch (error) {
            throw new Error(`Failed to prepare Docker image: ${error.message}`);
        }
    }

    async function deployConsumerToCloudRun(node, msg) {
        try {
            // Get the proxy URL from the incoming message
            const proxyUrl = msg.config && msg.config.proxyUrl 
                ? msg.config.proxyUrl 
                : (msg.proxyUrl || '');
            
            if (!proxyUrl) {
                throw new Error("Missing proxy URL. Make sure to run deploy-proxy first.");
            }

            const imageTag = node.consumerVersion;
            const imageName = `gcr.io/${node.projectId}/morpheus-lumerin-node:${imageTag}`;
            
            // Ensure service account has access to secrets
            const checkSaCmd = `gcloud auth list --filter=status:ACTIVE --format="value(account)"`;
            const { stdout: saResult } = await node._execAsync(checkSaCmd);
            const currentSA = saResult.trim();
            
            // Add IAM binding with the current service account
            const iamCmd = `gcloud projects add-iam-policy-binding ${node.projectId} --member="user:${currentSA}" --role="roles/secretmanager.secretAccessor"`;
            await node._execAsync(iamCmd);
            
            // Build the environment variables string for gcloud command
            const envVars = [
                `PROXY_ADDRESS=${node.proxyAddress}`,
                `WEB_ADDRESS=${node.webAddress}`,
                `WALLET_PRIVATE_KEY=${node.walletKey}`,
                `DIAMOND_CONTRACT_ADDRESS=${node.contractAddress}`,
                `MOR_TOKEN_ADDRESS=${node.morTokenAddress}`,
                `EXPLORER_API_URL=${node.explorerApiUrl}`,
                `ETH_NODE_CHAIN_ID=${node.ethNodeChainId}`,
                `ENVIRONMENT=${node.environment}`,
                `ETH_NODE_USE_SUBSCRIPTIONS=${node.ethNodeUseSubscriptions}`,
                `ETH_NODE_ADDRESS=${node.blockchainHttpUrl}`,
                `ETH_NODE_LEGACY_TX=${node.ethNodeLegacyTx}`,
                `PROXY_STORE_CHAT_CONTEXT=${node.proxyStoreChatContext}`,
                `PROXY_STORAGE_PATH=${node.proxyStoragePath}`,
                `LOG_COLOR=${node.logColor}`,
                `LOG_LEVEL=${node.logLevel || "info"}`,
                `LOG_FORMAT=${node.logFormat || "text"}`,
                `PROVIDER_CACHE_TTL=${node.providerCacheTtl || "60"}`,
                `MAX_CONCURRENT_SESSIONS=${node.maxConcurrentSessions || "100"}`,
                `SESSION_TIMEOUT=${node.sessionTimeout || "3600"}`,
                `CONSUMER_USERNAME=${node.consumerUsername}`,
                `CONSUMER_PASSWORD=${node.consumerPassword}`,
                `BLOCKCHAIN_WS_URL=${node.blockchainWsUrl}`,
                `BLOCKCHAIN_HTTP_URL=${node.blockchainHttpUrl}`,
                `BLOCKSCOUT_API_URL=${node.explorerApiUrl}`,
                `COOKIE_FILE_PATH=/secrets/.cookie`,
                `GO_ENV=production`
            ].join(',');
            
            // Deploy to Cloud Run with the cookie secret
            const deployCmd = `gcloud run deploy consumer-node \
                --image "${imageName}" \
                --platform managed \
                --region "${node.region}" \
                --allow-unauthenticated \
                --port=8082 \
                --set-secrets="/secrets/.cookie=${node.cookieSecretName}:latest" \
                --set-env-vars "${envVars}"`;
            
            await node._execAsync(deployCmd);
            
            // Wait for deployment and get service URL
            const { serviceUrl, serviceHealthy } = await checkDeployment(node, "consumer-node");
            
            if (!serviceUrl) {
                throw new Error("Failed to get consumer URL after deployment");
            }
            
            // Update consumer node with WEB_PUBLIC_URL
            const updateCmd = `gcloud run services update consumer-node \
                --region "${node.region}" \
                --platform managed \
                --update-env-vars "WEB_PUBLIC_URL=${serviceUrl}"`;
            
            await node._execAsync(updateCmd);
            
            return {
                consumerUrl: serviceUrl,
                output: `Deployed consumer node to ${serviceUrl}`
            };
        } catch (error) {
            throw new Error(`Deployment to Cloud Run failed: ${error.message}`);
        }
    }

    async function checkDeployment(node, serviceName) {
        try {
            const maxAttempts = 30;
            let attempt = 1;
            
            while (attempt <= maxAttempts) {
                const statusCmd = `gcloud run services describe ${serviceName} \
                    --region=${node.region} \
                    --format='value(status.conditions[0].status)' 2>/dev/null || echo "Unknown"`;
                
                const { stdout: status } = await node._execAsync(statusCmd);
                
                if (status.trim() === "True") {
                    // Get service URL
                    const urlCmd = `gcloud run services describe ${serviceName} \
                        --format 'value(status.url)' \
                        --region ${node.region}`;
                    
                    const { stdout: serviceUrl } = await node._execAsync(urlCmd);
                    return {
                        serviceUrl: serviceUrl.trim(),
                        serviceHealthy: true
                    };
                }
                
                node.status({fill:"blue", shape:"dot", text:`Waiting for deployment... (${attempt}/${maxAttempts})`});
                await new Promise(resolve => setTimeout(resolve, 10000)); // Wait 10 seconds
                attempt++;
            }
            
            throw new Error(`Deployment timed out after ${maxAttempts} attempts`);
        } catch (error) {
            throw new Error(`Failed to check deployment: ${error.message}`);
        }
    }

    async function checkConsumerHealth(node, consumerUrl) {
        try {
            const maxAttempts = 30;
            let attempt = 1;
            
            while (attempt <= maxAttempts) {
                try {
                    const { stdout } = await node._execAsync(`curl -s "${consumerUrl}/healthcheck"`);
                    
                    if (stdout && (stdout.includes("healthy") || stdout.includes("status"))) {
                        return true;
                    }
                } catch (err) {
                    // Ignore errors during health check attempts
                }
                
                node.status({fill:"blue", shape:"dot", text:`Checking health... (${attempt}/${maxAttempts})`});
                await new Promise(resolve => setTimeout(resolve, 10000)); // Wait 10 seconds
                attempt++;
            }
            
            throw new Error(`Health check failed after ${maxAttempts} attempts`);
        } catch (error) {
            throw new Error(`Health check failed: ${error.message}`);
        }
    }

    function handleDeploymentError(node, msg, error) {
        node.error(`Deployment error: ${error.message}`);
        node.status({fill:"red", shape:"dot", text: error.message});
        
        msg.payload = {
            status: 'error',
            action: 'deploy',
            error: error.message,
            output: error.stdout || '',
            stderr: error.stderr || ''
        };
        
        node.send(msg);
    }

    RED.nodes.registerType("deploy-consumer", ConsumerNode, {
        category: "Morpheus",
        color: "#a6bbcf",
        defaults: {
            name: { value: "" },
            // GCP Configuration
            projectId: { value: "", required: true },
            region: { value: "us-west1", required: true },
            dockerRegistry: { value: "srt0422", required: true },
            consumerVersion: { value: "v0.0.19", required: true },
            // Authentication Configuration
            consumerUsername: { value: "admin", required: true },
            consumerPassword: { value: "consumer-password-123", required: true },
            useCookieSecret: { value: true },
            cookieSecretName: { value: "COOKIE_SECRET" },
            // Blockchain Configuration
            walletKey: { value: "" },
            contractAddress: { value: "0xb8C55cD613af947E73E262F0d3C54b7211Af16CF" },
            morTokenAddress: { value: "0x34a285a1b1c166420df5b6630132542923b5b27e" },
            blockchainWsUrl: { value: "" },
            blockchainHttpUrl: { value: "https://sepolia-rollup.arbitrum.io/rpc" },
            explorerApiUrl: { value: "https://api-sepolia.arbiscan.io/api" },
            ethNodeChainId: { value: "421614" },
            ethNodeLegacyTx: { value: "false" },
            ethNodeUseSubscriptions: { value: "false" },
            // Service Configuration
            proxyAddress: { value: "0.0.0.0:3333" },
            webAddress: { value: "0.0.0.0:8082" },
            webPublicUrl: { value: "" },
            environment: { value: "development" },
            // Storage Configuration
            proxyStoreChatContext: { value: "true" },
            proxyStoragePath: { value: "./data/" },
            // Logging Configuration
            logLevel: { value: "info" },
            logFormat: { value: "text" },
            logColor: { value: "true" },
            // Performance Configuration
            providerCacheTtl: { value: "60" },
            maxConcurrentSessions: { value: "100" },
            sessionTimeout: { value: "3600" }
        },
        inputs: 1,
        outputs: 1,
        icon: "consumer.png",
        label: function() {
            return this.name || "Consumer";
        },
        paletteLabel: "Consumer"
    });
}; 