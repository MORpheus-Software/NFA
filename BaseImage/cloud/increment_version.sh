#!/bin/bash
ENV_FILE=".env"
REMOTE_API="https://hub.docker.com/v2/repositories/srt0422/openai-morpheus-proxy/tags?page_size=100"

# Fetch latest remote tags and filter semantic version tags
latest_remote_tag=$(curl -s "${REMOTE_API}" | jq -r '.results[].name' | grep '^v[0-9]\+\.[0-9]\+\.[0-9]\+$' | sort -V | tail -n 1)

if [ -z "$latest_remote_tag" ]; then
    echo "Failed to retrieve remote version tag. Defaulting to v0.0.0"
    latest_remote_tag="v0.0.0"
fi

echo "Latest remote version tag is ${latest_remote_tag}"

# Remove leading 'v' and split the version numbers
version_numbers=${latest_remote_tag#v}
IFS='.' read -r major minor patch <<< "$version_numbers"

# Increment the patch number by 1
new_patch=$((patch + 1))
new_version="v${major}.${minor}.${new_patch}"
echo "New version tag will be ${new_version}"

# Update (or insert) the NFA_PROXY_VERSION variable in the .env file
if grep -q '^NFA_PROXY_VERSION=' "$ENV_FILE"; then
    sed -i.bak "s/^NFA_PROXY_VERSION=.*/NFA_PROXY_VERSION=\"${new_version}\"/" "$ENV_FILE"
else
    # Append the variable if not present
    echo "NFA_PROXY_VERSION=\"${new_version}\"" >> "$ENV_FILE"
fi

echo "Environment file ${ENV_FILE} has been updated with new version tag ${new_version}" 