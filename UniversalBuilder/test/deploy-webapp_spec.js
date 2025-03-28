const helper = require('node-red-node-test-helper');
const deployWebapp = require('../nodes/deploy-webapp.js');
const { expect } = require('chai');
const sinon = require('sinon');

helper.init(require.resolve('node-red'));

describe('deploy-webapp Node', function() {
    // Increase timeout for deployment tests
    this.timeout(60000);

    before(function(done) {
        // Start server once before all tests
        helper.startServer(done);
    });

    after(function(done) {
        // Cleanup after all tests
        helper.unload().then(() => {
            helper.stopServer(done);
        });
    });

    beforeEach(function(done) {
        // Clear runtime between tests
        helper.unload().then(() => {
            done();
        });
    });

    it('should be loaded with correct defaults', async function() {
        const flow = [{
            id: "n1",
            type: "deploy-webapp",
            name: "test webapp"
        }];

        await helper.load(deployWebapp, flow);
        const n1 = helper.getNode("n1");
        expect(n1).to.have.property('name', 'test webapp');
        expect(n1).to.have.property('region', 'us-west1');
        expect(n1).to.have.property('dockerRegistry', 'srt0422');
        expect(n1).to.have.property('version', 'latest');
        expect(n1).to.have.property('modelName', 'Default Model');
        expect(n1).to.have.property('chatCompletionsPath', '/v1/chat/completions');
    });

    it('should validate webapp configuration', async function() {
        const flow = [{
            id: "n1",
            type: "deploy-webapp",
            name: "test webapp",
            projectId: "test-project",
            region: "us-central1",
            action: "deploy",
            modelName: "gpt-4"
        }];

        await helper.load(deployWebapp, flow);
        const n1 = helper.getNode("n1");
        expect(n1).to.have.property('projectId', 'test-project');
        expect(n1).to.have.property('region', 'us-central1');
        expect(n1).to.have.property('action', 'deploy');
        expect(n1).to.have.property('modelName', 'gpt-4');
    });

    it('should handle deployment with proxy URL from previous node', async function() {
        const flow = [
            {
                id: "n1",
                type: "deploy-webapp",
                name: "test webapp",
                projectId: "test-project",
                region: "us-central1",
                wires: [["n2"]]
            },
            { id: "n2", type: "helper" }
        ];

        // Create a stub for execAsync
        const execStub = sinon.stub();
        
        // Success responses for different command patterns
        execStub.withArgs(sinon.match(/gcloud config set project/)).resolves({ stdout: "", stderr: "" });
        execStub.withArgs(sinon.match(/gcloud auth print-access-token/)).resolves({ stdout: "fake-token", stderr: "" });
        execStub.withArgs(sinon.match(/docker manifest inspect/)).resolves({ stdout: "image exists", stderr: "" });
        execStub.withArgs(sinon.match(/gcloud run deploy chat-web-app/)).resolves({ stdout: "Deployment successful", stderr: "" });
        execStub.withArgs(sinon.match(/gcloud run services describe chat-web-app.*status\.conditions/)).resolves({ stdout: "True\nReady", stderr: "" });
        execStub.withArgs(sinon.match(/gcloud run services describe chat-web-app.*status\.url/)).resolves({ stdout: "https://chat-web-app-test.run.app", stderr: "" });

        await helper.load(deployWebapp, flow);
        const n2 = helper.getNode("n2");
        const n1 = helper.getNode("n1");
        
        // Replace the execAsync function with our stub
        n1._execAsync = execStub;

        return new Promise((resolve) => {
            n2.on("input", function(msg) {
                try {
                    expect(msg).to.have.property('payload');
                    expect(msg.payload).to.have.property('status', 'success');
                    expect(msg.payload).to.have.property('webappUrl', 'https://chat-web-app-test.run.app');
                    expect(msg.config).to.have.property('webappUrl', 'https://chat-web-app-test.run.app');
                    // Check if the command used the right proxyUrl
                    const deployCallArgs = execStub.getCalls().find(call => 
                        call.args[0].includes('gcloud run deploy')
                    ).args[0];
                    expect(deployCallArgs).to.include('OPENAI_API_URL=https://test-proxy-url.run.app/v1');
                    resolve();
                } catch (err) {
                    resolve(err);
                }
            });
            
            // Send a message with proxyUrl from previous deployment
            n1.receive({
                config: {
                    proxyUrl: 'https://test-proxy-url.run.app',
                    region: 'us-central1',
                    projectId: 'test-project'
                }
            });
        });
    });

    it('should handle webapp update action', async function() {
        const flow = [
            {
                id: "n1",
                type: "deploy-webapp",
                name: "test webapp",
                projectId: "test-project",
                region: "us-central1",
                action: "update",
                wires: [["n2"]]
            },
            { id: "n2", type: "helper" }
        ];

        // Create a stub for execAsync
        const execStub = sinon.stub();
        
        // Success responses for different command patterns
        execStub.withArgs(sinon.match(/gcloud config set project/)).resolves({ stdout: "", stderr: "" });
        execStub.withArgs(sinon.match(/gcloud auth print-access-token/)).resolves({ stdout: "fake-token", stderr: "" });
        execStub.withArgs(sinon.match(/docker manifest inspect/)).resolves({ stdout: "image exists", stderr: "" });
        execStub.withArgs(sinon.match(/gcloud run update chat-web-app/)).resolves({ stdout: "Update successful", stderr: "" });
        execStub.withArgs(sinon.match(/gcloud run services describe chat-web-app.*status\.conditions/)).resolves({ stdout: "True\nReady", stderr: "" });
        execStub.withArgs(sinon.match(/gcloud run services describe chat-web-app.*status\.url/)).resolves({ stdout: "https://chat-web-app-test.run.app", stderr: "" });

        await helper.load(deployWebapp, flow);
        const n2 = helper.getNode("n2");
        const n1 = helper.getNode("n1");
        
        // Replace the execAsync function with our stub
        n1._execAsync = execStub;

        return new Promise((resolve) => {
            n2.on("input", function(msg) {
                try {
                    expect(msg.payload).to.have.property('action', 'update');
                    expect(msg.payload).to.have.property('status', 'success');
                    resolve();
                } catch (err) {
                    resolve(err);
                }
            });
            
            n1.receive({
                config: {
                    proxyUrl: 'https://test-proxy-url.run.app',
                    region: 'us-central1',
                    projectId: 'test-project'
                }
            });
        });
    });

    it('should wait for proxy URL if not available', async function() {
        const flow = [
            {
                id: "n1",
                type: "deploy-webapp",
                name: "test webapp",
                projectId: "test-project",
                region: "us-central1",
                wires: [["n2"]]
            },
            { id: "n2", type: "helper" }
        ];

        await helper.load(deployWebapp, flow);
        const n2 = helper.getNode("n2");
        const n1 = helper.getNode("n1");

        return new Promise((resolve) => {
            n2.on("input", function(msg) {
                try {
                    expect(msg.payload).to.have.property('status', 'pending');
                    expect(msg.payload).to.have.property('message').that.includes('Waiting for proxy URL');
                    resolve();
                } catch (err) {
                    resolve(err);
                }
            });
            
            // Send message without proxy URL
            n1.receive({
                config: {
                    region: 'us-central1',
                    projectId: 'test-project'
                }
            });
        });
    });

    it('should handle deployment errors gracefully', async function() {
        const flow = [
            {
                id: "n1",
                type: "deploy-webapp",
                name: "test webapp",
                wires: [["n2"]]
            },
            { id: "n2", type: "helper" }
        ];

        await helper.load(deployWebapp, flow);
        const n2 = helper.getNode("n2");
        const n1 = helper.getNode("n1");

        return new Promise((resolve) => {
            n2.on("input", function(msg) {
                try {
                    expect(msg.payload).to.have.property('error');
                    expect(msg.payload).to.have.property('status', 'error');
                    resolve();
                } catch (err) {
                    resolve(err);
                }
            });
            // Send message without required config
            n1.receive({payload: "test"});
        });
    });
}); 