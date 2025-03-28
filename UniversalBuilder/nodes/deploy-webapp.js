const { exec } = require('child_process');
const util = require('util');
const execAsync = util.promisify(exec);
const https = require('https');
const http = require('http');

module.exports = function(RED) {
    function WebAppNode(config) {
        RED.nodes.createNode(this, config);
        const node = this;

        // Store all configuration values with proper validation
        this.name = config.name;
        this.action = config.action || 'deploy';
        this.projectId = config.projectId;
        this.region = config.region || "us-west1";
        this.dockerRegistry = config.dockerRegistry || "srt0422";
        this.version = config.version || "latest";
        this.openaiApiUrl = config.openaiApiUrl || "";
        this.modelName = config.modelName || "Default Model";
        this.chatCompletionsPath = config.chatCompletionsPath || "/v1/chat/completions";

        // Allow exec function injection for testing
        this._execAsync = execAsync;

        // Validate required configuration
        function validateConfig(config) {
            const required = [
                'projectId', 
                'region'
            ];
            const missing = required.filter(field => !config[field]);
            if (missing.length > 0) {
                throw new Error(`Missing required configuration: ${missing.join(', ')}`);
            }
        }
        
        // Handle deployment errors
        function handleError(err, msg) {
            node.error(err);
            node.status({fill:"red",shape:"dot",text:err.message});
            msg.payload = {
                error: err.message || err,
                status: 'error',
                action: 'deploy'
            };
            return msg;
        }

        // Check deployment status
        async function checkDeployment(serviceName, region) {
            try {
                const cmd = `gcloud run services describe ${serviceName} --region ${region} --format 'get(status.conditions[0].status,status.conditions[0].message)'`;
                const { stdout } = await node._execAsync(cmd);
                const [status, message] = stdout.trim().split('\n');
                return status === 'True';
            } catch (err) {
                throw new Error(`Failed to check deployment status: ${err.message}`);
            }
        }

        // Get service URL
        async function getServiceUrl(serviceName, region) {
            try {
                const cmd = `gcloud run services describe ${serviceName} --region ${region} --format 'get(status.url)'`;
                const { stdout } = await node._execAsync(cmd);
                return stdout.trim();
            } catch (err) {
                throw new Error(`Failed to get service URL: ${err.message}`);
            }
        }

        // Check service health
        async function checkServiceHealth(url) {
            return new Promise((resolve) => {
                const protocol = url.startsWith('https') ? https : http;
                
                const req = protocol.get(url, (res) => {
                    resolve(res.statusCode >= 200 && res.statusCode < 300);
                });
                
                req.on('error', () => {
                    resolve(false);
                });
                
                req.setTimeout(5000, () => {
                    req.destroy();
                    resolve(false);
                });
            });
        }

        // Ensure GCP context
        async function ensureGcpContext(config) {
            try {
                // Set project ID
                await node._execAsync(`gcloud config set project ${config.projectId}`);
                
                // Check authentication
                try {
                    await node._execAsync('gcloud auth print-access-token');
                } catch (err) {
                    throw new Error('GCP authentication required. Please run "gcloud auth login" first.');
                }
            } catch (err) {
                throw new Error('Failed to set GCP project: ' + err.message);
            }
        }

        node.on('input', async function(msg) {
            try {
                const msgConfig = msg.config || {};
                
                // Merge configuration with message config taking precedence
                const effectiveConfig = {
                    projectId: msgConfig.projectId || this.projectId,
                    region: msgConfig.region || this.region,
                    dockerRegistry: msgConfig.dockerRegistry || this.dockerRegistry,
                    version: msgConfig.version || this.version,
                    openaiApiUrl: msgConfig.OPENAI_API_URL || msgConfig.openaiApiUrl || this.openaiApiUrl,
                    modelName: msgConfig.modelName || this.modelName,
                    chatCompletionsPath: msgConfig.chatCompletionsPath || this.chatCompletionsPath,
                    proxyUrl: msgConfig.proxyUrl || '',
                    consumerUrl: msgConfig.consumerUrl || ''
                };
                
                // Validate configuration before proceeding
                validateConfig(effectiveConfig);
                
                // Ensure we have the proxyUrl from previous deployment steps
                if (!effectiveConfig.proxyUrl) {
                    const proxyUrlFromConfig = msgConfig.OPENAI_API_URL || msgConfig.proxyUrl || '';
                    
                    if (proxyUrlFromConfig) {
                        effectiveConfig.proxyUrl = proxyUrlFromConfig;
                    } else {
                        node.warn('No proxy URL found. Make sure deploy-proxy has been run first.');
                        msg.payload = {
                            status: 'pending',
                            action: 'deploy',
                            message: 'Waiting for proxy URL before deploying'
                        };
                        node.send(msg);
                        return;
                    }
                }

                // Set node status
                node.status({fill:"blue", shape:"dot", text:"Deploying webapp..."});
                
                // Ensure GCP context
                await ensureGcpContext(effectiveConfig);
                
                // Determine if we're updating or deploying
                const deployCmd = this.action === 'update' ? 'update' : 'deploy';
                
                // Define the Docker image to use
                const imageName = `${effectiveConfig.dockerRegistry}/chat-web-app:${effectiveConfig.version}`;
                const serviceName = 'chat-web-app';
                
                // Check if Docker Hub image exists
                let checkImageCmd = `docker manifest inspect ${imageName} > /dev/null 2>&1 || echo "Image not found"`;
                const { stdout: checkImageResult } = await node._execAsync(checkImageCmd);
                const imageExists = !checkImageResult.includes('Image not found');
                
                if (!imageExists) {
                    node.warn(`Image ${imageName} not found. Using a default image.`);
                }

                // Define chat completions path with fallback
                const chatCompletionsPath = effectiveConfig.chatCompletionsPath || "/v1/chat/completions";
                
                // Deploy to Cloud Run 
                node.status({fill:"blue", shape:"dot", text:`${deployCmd}ing webapp...`});
                const deployWebAppCmd = `gcloud run ${deployCmd} ${serviceName} \
                    --image "${imageName}" \
                    --platform managed \
                    --region "${effectiveConfig.region}" \
                    --allow-unauthenticated \
                    --set-env-vars "\
OPENAI_API_URL=${effectiveConfig.proxyUrl}/v1,\
CHAT_COMPLETIONS_PATH=${chatCompletionsPath},\
NEXT_PUBLIC_CHAT_COMPLETIONS_PATH=${chatCompletionsPath},\
MODEL_NAME=${effectiveConfig.modelName}"`;
                
                await node._execAsync(deployWebAppCmd);
                
                // Wait for deployment to complete
                node.status({fill:"blue", shape:"dot", text:"Checking deployment..."});
                let deployed = false;
                for (let i = 0; i < 30 && !deployed; i++) {
                    deployed = await checkDeployment(serviceName, effectiveConfig.region);
                    if (!deployed) await new Promise(resolve => setTimeout(resolve, 2000));
                }
                
                if (!deployed) {
                    throw new Error('Deployment timed out');
                }

                // Get the service URL
                const webappUrl = await getServiceUrl(serviceName, effectiveConfig.region);
                
                // Check service health
                node.status({fill:"blue", shape:"dot", text:"Checking health..."});
                const isHealthy = await checkServiceHealth(webappUrl);
                
                if (!isHealthy) {
                    node.warn('Web app deployed but health check failed.');
                }
                
                // Pass configuration to next node
                msg.config = {
                    ...msg.config,
                    webappUrl
                };
                
                // Update payload with deployment results
                msg.payload = {
                    status: 'success',
                    action: this.action,
                    webappUrl,
                    message: `Web app successfully ${this.action === 'update' ? 'updated' : 'deployed'}`
                };
                
                node.status({fill:"green", shape:"dot", text:`Deployed: ${webappUrl}`});
                node.send(msg);
            } catch (error) {
                node.status({fill:"red", shape:"dot", text:"Deployment failed"});
                msg = handleError(error, msg);
                node.send(msg);
            }
        });
    }

    RED.nodes.registerType("deploy-webapp", WebAppNode, {
        category: "Morpheus",
        color: "#a6bbcf",
        defaults: {
            name: { value: "" },
            action: { value: "deploy" },
            projectId: { required: true },
            region: { value: "us-west1", required: true },
            dockerRegistry: { value: "srt0422", required: true },
            version: { value: "latest", required: true },
            openaiApiUrl: { value: "", required: false },
            modelName: { value: "Default Model", required: true },
            chatCompletionsPath: { value: "/v1/chat/completions", required: false }
        },
        inputs: 1,
        outputs: 1,
        icon: "white-globe.svg",
        label: function() {
            return this.name || "Web App";
        },
        paletteLabel: "Web App"
    });
} 