module.exports = function(RED) {
    function DeployCloudRunNode(config) {
        RED.nodes.createNode(this, config);
        var node = this;
        
        // Store Cloud Run specific configuration
        this.name = config.name;
        this.project = config.project;
        this.region = config.region;
        this.service = config.service;

        node.on('input', function(msg, send, done) {
            // Pass through execution
            send(msg);
            if (done) {
                done();
            }
        });
    }
    RED.nodes.registerType("deploy-cloud-run", DeployCloudRunNode);
}