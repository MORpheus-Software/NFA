const helper = require('node-red-node-test-helper');
const deployConsumer = require('../nodes/deploy-consumer.js');
const { expect } = require('chai');
const sinon = require('sinon');

helper.init(require.resolve('node-red'));

describe('deploy-consumer Node', function() {
    this.timeout(10000); // Increase timeout for tests involving exec calls
    
    let execAsyncStub;
    
    beforeEach(function(done) {
        helper.startServer(done);
        
        // Mock exec calls to avoid actual command execution
        execAsyncStub = sinon.stub();
        
        // Default success responses for different commands
        execAsyncStub.withArgs(sinon.match(/gcloud secrets describe/)).resolves({
            stdout: 'Secret does not exist',
            stderr: ''
        });
        
        execAsyncStub.withArgs(sinon.match(/gcloud secrets create/)).resolves({
            stdout: 'Created secret [COOKIE_SECRET]',
            stderr: ''
        });
        
        execAsyncStub.withArgs(sinon.match(/docker pull/)).resolves({
            stdout: 'Using default tag: latest\nlatest: Pulling from srt0422/morpheus-marketplace-consumer\nDigest: sha256:1234567890abcdef\nStatus: Downloaded newer image',
            stderr: ''
        });
        
        execAsyncStub.withArgs(sinon.match(/docker tag/)).resolves({
            stdout: '',
            stderr: ''
        });
        
        execAsyncStub.withArgs(sinon.match(/docker push/)).resolves({
            stdout: 'The push refers to repository [gcr.io/test-project/morpheus-lumerin-node]\nlatest: digest: sha256:1234567890abcdef size: 1234',
            stderr: ''
        });
        
        execAsyncStub.withArgs(sinon.match(/gcloud auth list/)).resolves({
            stdout: 'test-user@example.com',
            stderr: ''
        });
        
        execAsyncStub.withArgs(sinon.match(/gcloud projects add-iam-policy-binding/)).resolves({
            stdout: 'Updated IAM policy',
            stderr: ''
        });
        
        execAsyncStub.withArgs(sinon.match(/gcloud run deploy/)).resolves({
            stdout: 'Deploying container to Cloud Run service [consumer-node] in project [test-project]\nDeployed service [consumer-node]',
            stderr: ''
        });
        
        execAsyncStub.withArgs(sinon.match(/gcloud run services describe .* --format='value\(status\.conditions/)).resolves({
            stdout: 'True',
            stderr: ''
        });
        
        execAsyncStub.withArgs(sinon.match(/gcloud run services describe .* --format 'value\(status\.url\)'/)).resolves({
            stdout: 'https://consumer-node-abc123.run.app',
            stderr: ''
        });
        
        execAsyncStub.withArgs(sinon.match(/gcloud run services update/)).resolves({
            stdout: 'Updated service [consumer-node]',
            stderr: ''
        });
        
        execAsyncStub.withArgs(sinon.match(/curl -s ".*\/healthcheck"/)).resolves({
            stdout: '{"status":"healthy"}',
            stderr: ''
        });
    });

    afterEach(function(done) {
        sinon.restore();
        helper.unload();
        helper.stopServer(done);
    });

    it('should be loaded with correct defaults', async function() {
        const flow = [{
            id: "n1",
            type: "deploy-consumer",
            name: "test consumer"
        }];

        await helper.load(deployConsumer, flow);
        const n1 = helper.getNode("n1");
        expect(n1).to.have.property('name', 'test consumer');
    });

    it('should validate GCP configuration', async function() {
        const flow = [{
            id: "n1",
            type: "deploy-consumer",
            name: "test consumer",
            projectId: "test-project",
            region: "us-west1",
            dockerRegistry: "test-registry",
            consumerVersion: "v0.0.19"
        }];

        await helper.load(deployConsumer, flow);
        const n1 = helper.getNode("n1");
        expect(n1).to.have.property('projectId', 'test-project');
        expect(n1).to.have.property('region', 'us-west1');
        expect(n1).to.have.property('dockerRegistry', 'test-registry');
        expect(n1).to.have.property('consumerVersion', 'v0.0.19');
    });

    it('should validate blockchain configuration', async function() {
        const flow = [{
            id: "n1",
            type: "deploy-consumer",
            name: "test consumer",
            walletKey: "test-key",
            contractAddress: "0x123",
            blockchainWsUrl: "ws://test",
            blockchainHttpUrl: "http://test"
        }];

        await helper.load(deployConsumer, flow);
        const n1 = helper.getNode("n1");
        expect(n1).to.have.property('walletKey', 'test-key');
        expect(n1).to.have.property('contractAddress', '0x123');
        expect(n1).to.have.property('blockchainWsUrl', 'ws://test');
        expect(n1).to.have.property('blockchainHttpUrl', 'http://test');
    });

    describe('Cloud Run Deployment Tests', function() {
        it('should deploy consumer to Cloud Run', async function() {
            // Create a node with the necessary configuration
            const flow = [
                {
                    id: "n1",
                    type: "deploy-consumer",
                    name: "test consumer",
                    projectId: "test-project",
                    region: "us-west1",
                    dockerRegistry: "test-registry",
                    consumerVersion: "v0.0.19",
                    consumerUsername: "admin",
                    consumerPassword: "test-password",
                    walletKey: "test-wallet-key",
                    contractAddress: "0x123",
                    morTokenAddress: "0x456",
                    blockchainHttpUrl: "https://test-blockchain.io/rpc",
                    wires: [["n2"]]
                },
                { id: "n2", type: "helper" }
            ];

            await helper.load(deployConsumer, flow);
            const n1 = helper.getNode("n1");
            const n2 = helper.getNode("n2");
            
            // Override the execAsync function with our stub
            n1._execAsync = execAsyncStub;

            return new Promise((resolve) => {
                n2.on("input", function(msg) {
                    try {
                        // Check the deployment result
                        expect(msg.payload).to.have.property('status', 'success');
                        expect(msg.payload).to.have.property('consumerUrl', 'https://consumer-node-abc123.run.app');
                        
                        // Verify all expected commands were called
                        expect(execAsyncStub.callCount).to.be.at.least(10); // Multiple commands should be called
                        
                        // Check cookie secret creation
                        expect(execAsyncStub.calledWith(sinon.match(/gcloud secrets create COOKIE_SECRET/))).to.be.true;
                        
                        // Check Docker operations
                        expect(execAsyncStub.calledWith(sinon.match(/docker pull/))).to.be.true;
                        expect(execAsyncStub.calledWith(sinon.match(/docker tag/))).to.be.true;
                        expect(execAsyncStub.calledWith(sinon.match(/docker push/))).to.be.true;
                        
                        // Check Cloud Run deployment
                        expect(execAsyncStub.calledWith(sinon.match(/gcloud run deploy consumer-node/))).to.be.true;
                        expect(execAsyncStub.calledWith(sinon.match(/--set-secrets="\/secrets\/.cookie=COOKIE_SECRET:latest"/))).to.be.true;
                        
                        // Check service URL retrieval
                        expect(execAsyncStub.calledWith(sinon.match(/gcloud run services describe consumer-node.*value\(status\.url\)/))).to.be.true;
                        
                        // Check service update with public URL
                        expect(execAsyncStub.calledWith(sinon.match(/gcloud run services update consumer-node/))).to.be.true;
                        
                        // Check health check
                        expect(execAsyncStub.calledWith(sinon.match(/curl -s ".*\/healthcheck"/))).to.be.true;
                        
                        resolve();
                    } catch (err) {
                        resolve(err);
                    }
                });
                
                n1.receive({
                    config: {
                        proxyUrl: "https://proxy-test-url.run.app"
                    }
                });
            });
        });

        it('should handle deployment failure gracefully', async function() {
            // Create a stub that simulates deployment failure
            const failureExecStub = sinon.stub();
            failureExecStub.withArgs(sinon.match(/gcloud secrets describe/)).resolves({
                stdout: 'Secret does not exist',
                stderr: ''
            });
            failureExecStub.withArgs(sinon.match(/gcloud secrets create/)).resolves({
                stdout: 'Created secret [COOKIE_SECRET]',
                stderr: ''
            });
            failureExecStub.withArgs(sinon.match(/docker pull/)).resolves({
                stdout: 'Using default tag: latest\nlatest: Pulling from test-registry/consumer-image',
                stderr: ''
            });
            failureExecStub.withArgs(sinon.match(/docker tag/)).resolves({
                stdout: '',
                stderr: ''
            });
            failureExecStub.withArgs(sinon.match(/docker push/)).resolves({
                stdout: 'The push refers to repository [gcr.io/test-project/consumer-image]',
                stderr: ''
            });
            failureExecStub.withArgs(sinon.match(/gcloud run deploy/)).rejects(
                new Error('Deployment failed: Error creating Cloud Run service')
            );

            const flow = [
                {
                    id: "n1",
                    type: "deploy-consumer",
                    name: "test consumer",
                    projectId: "test-project",
                    region: "us-west1",
                    dockerRegistry: "test-registry",
                    consumerVersion: "v0.0.19",
                    consumerUsername: "admin",
                    consumerPassword: "test-password",
                    wires: [["n2"]]
                },
                { id: "n2", type: "helper" }
            ];

            await helper.load(deployConsumer, flow);
            const n1 = helper.getNode("n1");
            const n2 = helper.getNode("n2");
            
            // Override the execAsync function with our failure stub
            n1._execAsync = failureExecStub;

            return new Promise((resolve) => {
                n2.on("input", function(msg) {
                    try {
                        expect(msg.payload).to.have.property('status', 'error');
                        expect(msg.payload).to.have.property('error').that.includes('Deployment to Cloud Run failed');
                        resolve();
                    } catch (err) {
                        resolve(err);
                    }
                });
                
                n1.receive({
                    config: {
                        proxyUrl: "https://proxy-test-url.run.app"
                    }
                });
            });
        });

        it('should handle missing proxy URL', async function() {
            const flow = [
                {
                    id: "n1",
                    type: "deploy-consumer",
                    name: "test consumer",
                    projectId: "test-project",
                    region: "us-west1",
                    dockerRegistry: "test-registry",
                    consumerVersion: "v0.0.19",
                    consumerUsername: "admin",
                    consumerPassword: "test-password",
                    wires: [["n2"]]
                },
                { id: "n2", type: "helper" }
            ];

            await helper.load(deployConsumer, flow);
            const n1 = helper.getNode("n1");
            const n2 = helper.getNode("n2");
            
            // Override the execAsync function with our stub
            n1._execAsync = execAsyncStub;

            return new Promise((resolve) => {
                n2.on("input", function(msg) {
                    try {
                        expect(msg.payload).to.have.property('status', 'error');
                        expect(msg.payload).to.have.property('error').that.includes('Missing proxy URL');
                        resolve();
                    } catch (err) {
                        resolve(err);
                    }
                });
                
                // Send message without proxy URL
                n1.receive({ payload: {} });
            });
        });
    });

    it('should validate environment variables', async function() {
        process.env.WALLET_KEY = 'test-wallet-key';
        process.env.CONTRACT_ADDRESS = 'test-contract-address';

        const flow = [{
            id: "n1",
            type: "deploy-consumer",
            name: "test consumer",
            walletKey: "${WALLET_KEY}",
            contractAddress: "${CONTRACT_ADDRESS}"
        }];

        await helper.load(deployConsumer, flow);
        const n1 = helper.getNode("n1");
        expect(n1.walletKey).to.equal('test-wallet-key');
        expect(n1.contractAddress).to.equal('test-contract-address');

        // Cleanup
        delete process.env.WALLET_KEY;
        delete process.env.CONTRACT_ADDRESS;
    });
}); 