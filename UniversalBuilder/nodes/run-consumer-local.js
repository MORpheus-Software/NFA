const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

module.exports = function(RED) {
    function RunConsumerLocalNode(config) {
        RED.nodes.createNode(this, config);
        const node = this;
        
        // Store configuration
        this.name = config.name;
        this.consumerPort = config.consumerPort || 3333;
        this.consumerImage = config.consumerImage || 'consumer:latest';
        
        let containerProcess = null;
        
        node.on('input', function(msg, send, done) {
            // Extract configuration from message or use defaults
            const consumerPort = msg.consumerPort || node.consumerPort;
            const consumerImage = msg.consumerImage || node.consumerImage;
            
            try {
                // Build docker run command
                const dockerArgs = [
                    'run', 
                    '-d',
                    '--name', `consumer-local-${Date.now()}`,
                    '-p', `${consumerPort}:3333`,
                    consumerImage
                ];
                
                // Add environment variables if provided
                if (msg.env && typeof msg.env === 'object') {
                    Object.entries(msg.env).forEach(([key, value]) => {
                        dockerArgs.push('-e', `${key}=${value}`);
                    });
                }
                
                // Run the container
                node.status({fill: "blue", shape: "dot", text: "Starting consumer container..."});
                containerProcess = spawn('docker', dockerArgs);
                
                containerProcess.stdout.on('data', (data) => {
                    const containerId = data.toString().trim();
                    node.status({fill: "green", shape: "dot", text: `Running: ${containerId.substring(0, 12)}`});
                    msg.containerId = containerId;
                    send(msg);
                });
                
                containerProcess.stderr.on('data', (data) => {
                    node.error(`Error starting consumer container: ${data.toString()}`);
                    node.status({fill: "red", shape: "ring", text: "Failed to start"});
                    if (done) done(new Error(data.toString()));
                });
                
                containerProcess.on('close', (code) => {
                    if (code !== 0) {
                        node.error(`Docker process exited with code ${code}`);
                        node.status({fill: "red", shape: "ring", text: `Error: exit code ${code}`});
                        if (done && !containerProcess.stdout.listenerCount('data')) done(new Error(`Process exited with code ${code}`));
                    } else if (done && !containerProcess.stdout.listenerCount('data')) {
                        done();
                    }
                });
            } catch (err) {
                node.error(err);
                node.status({fill: "red", shape: "ring", text: "Error"});
                if (done) done(err);
            }
        });
        
        node.on('close', function() {
            // Cleanup when node is removed or redeployed
            if (containerProcess) {
                containerProcess.kill();
            }
            node.status({});
        });
    }
    
    RED.nodes.registerType("run-consumer-local", RunConsumerLocalNode, {
        defaults: {
            name: { value: "" },
            consumerPort: { value: 3333, validate: RED.validators.number() },
            consumerImage: { value: "consumer:latest" }
        }
    });
} 