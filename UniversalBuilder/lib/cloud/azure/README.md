# Running Morpheus in Azure

### Prerequisites

- [azure-cli](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli)
- [terraform](https://developer.hashicorp.com/terraform/tutorials/aws-get-started/install-cli)
- [DockerHub PAT](https://docs.docker.com/security/for-developers/access-tokens/)

### Setup

To start, get the subscription ID for your Azure account:
```
az login
az account show --query id --output tsv
```

Then allow authentication with Azure, we can create a service principal:
```
az ad sp create-for-rbac --name "morpheus-terraform-sp" --role="Contributor" --scopes="/subscriptions/<AZURE_SUBSCRIPTION_ID>"
```

This should return a set of values:
- `appId` → Client ID
- `password` → Client Secret
- `tenant` → Tenant ID

Finally you can enable Terraform to authenticate with Azure. The following environment variables should be set locally or within a GitHub Action:
```
export ARM_CLIENT_ID="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
export ARM_CLIENT_SECRET="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
export ARM_SUBSCRIPTION_ID="<AZURE_SUBSCRIPTION_ID>"
export ARM_TENANT_ID="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

To enable Terraform to see your DockerHub PAT, set the following:
```
export TF_VAR_docker_username="username"
export TF_VAR_docker_password="access_token"
```

Before running the Terraform deployment, you may also want to verify the environment variables defined in `variables.tf`. To override any default values, simply add them to the associated `.tfvars` file for the environment in question.

If you'd like to tear down the stack, `terraform destroy -var-file=development.tfvars` will remove any previously created resources.

### Deployment

With the environment variables set, you can proceed with deploying the Morpheus stack into Azure using Terraform. 

You can either initiate Terraform locally or within a GitHub workflow. Be sure to pass the appropriate `.tfvars` file based on the environment you are deploying. 

Local:
```
terraform init
terraform plan -var-file=development.tfvars -out=tfplan
terraform apply -var-file=development.tfvars tfplan
```

Note: A [sample GitHub workflow](morpheus-azure-deployment.yaml) to deploy this Terraform stack can be found in the repo. 

After a successful plan/apply, you should see an endpoint and ports returned where the Morpheus application stack is now running!

### Overriding Default Variables

A default set of variables for the Morpheus stack are defined in [variables.tf](variables.tf).

If you would like to override any of these default values, or if you would like unique values in different environments, you can set alternate values in [development.tfvars](development.tfvars) and/or [production.tfvars](production.tfvars). Changing the defaults defined in [variables.tf](variables.tf) is not required.

##### Overriding a string/number/bool 

Simply set the same key with a different value in your `.tfvars` file:

```
environment = "development"
```

##### Overriding a single environment variable

Utilize the relevant `*_env_overrides` variable. These overrides will be merged with the other env defaults defined in `variables.tf`.

To override a web_app environment variable for example, set the following in your `.tfvars` file:
```
web_app_env_overrides = {
    MODEL_NAME = "Llama 3.2 3B Instruct" 
}
```

As mentioned above, these files will then be referenced when deploying the stack with `terraform plan` and `terraform apply`. 