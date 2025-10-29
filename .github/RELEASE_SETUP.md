# Release Setup Guide

This document provides instructions for configuring the repository to enable automated releases using release-please.

## Overview

The repository uses [release-please](https://github.com/googleapis/release-please) to automate releases. When code is merged to the `main` branch, release-please creates or updates a release pull request that tracks changes and version bumps according to [Conventional Commits](https://www.conventionalcommits.org/).

## Required Configuration

To enable release-please to create pull requests, you must configure ONE of the following options:

### Option 1: Enable GitHub Actions Permissions (Recommended)

This is the simplest approach and is recommended for most repositories.

1. Navigate to your repository on GitHub
2. Go to **Settings** → **Actions** → **General**
3. Scroll down to **Workflow permissions**
4. Select **"Read and write permissions"**
5. Check the box for **"Allow GitHub Actions to create and approve pull requests"**
6. Click **Save**

**Pros:**
- Simple to configure
- No additional secrets to manage
- Works automatically for all workflows

**Cons:**
- Gives all GitHub Actions workflows the ability to create PRs
- Some organizations may have policies against this setting

### Option 2: Use a Personal Access Token (PAT)

If you cannot enable the repository setting (e.g., due to organization policies), you can use a Personal Access Token.

#### Steps to create and configure a PAT:

1. **Create a Personal Access Token:**
   - Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
   - Click "Generate new token (classic)"
   - Give it a descriptive name (e.g., "release-please-bot")
   - Set an appropriate expiration time
   - Select the following scopes:
     - `repo` (Full control of private repositories)
     - `workflow` (Update GitHub Action workflows)
   - Click "Generate token" and copy the token value

2. **Add the token as a repository secret:**
   - Navigate to your repository on GitHub
   - Go to **Settings** → **Secrets and variables** → **Actions**
   - Click "New repository secret"
   - Name: `RELEASE_PLEASE_TOKEN`
   - Value: Paste your PAT
   - Click "Add secret"

3. **Update the workflow file:**
   - Edit `.github/workflows/release-please.yml`
   - In the "Run release-please" step, change:
     ```yaml
     token: ${{ secrets.GITHUB_TOKEN }}
     ```
     to:
     ```yaml
     token: ${{ secrets.RELEASE_PLEASE_TOKEN }}
     ```

**Pros:**
- Fine-grained control over permissions
- Can be used when repository settings cannot be changed
- Token can be scoped to a specific bot account

**Cons:**
- Requires creating and managing a PAT
- PAT needs to be rotated when it expires
- Additional secret to secure

### Option 3: Use a GitHub App Token

For organizations with strict security requirements, a GitHub App provides the most secure and fine-grained approach.

This is an advanced configuration. See the [release-please documentation](https://github.com/googleapis/release-please?tab=readme-ov-file#github-app-authentication) for details.

## Verifying the Configuration

After configuring one of the options above:

1. Make a commit to the `main` branch following [Conventional Commits](https://www.conventionalcommits.org/) format
   - Example: `feat: add new feature` or `fix: resolve bug`
2. Push the commit to trigger the release-please workflow
3. Check the Actions tab to verify the workflow runs successfully
4. A pull request should be created or updated with the release notes

## Troubleshooting

### Error: "GitHub Actions is not permitted to create or approve pull requests"

This error indicates that neither Option 1 nor Option 2 has been properly configured.

**Solution:**
- If using Option 1: Verify that "Allow GitHub Actions to create and approve pull requests" is checked in repository settings
- If using Option 2: Verify that the PAT secret is correctly configured and the workflow is using it

### Release PR is not being created

**Possible causes:**
1. Commits don't follow Conventional Commits format
2. No commits have been made since the last release
3. Workflow permissions are insufficient

**Solution:**
- Review recent commits and ensure they follow the format: `type: description`
- Check the Actions tab for workflow run logs
- Verify the configuration steps above

### Workflow runs but nothing happens

If the workflow completes successfully but no PR is created, it may be because:
- No releasable changes have been made (only chore commits, docs, etc.)
- The changes have already been included in an existing release PR

## Additional Resources

- [Release Please Documentation](https://github.com/googleapis/release-please)
- [Conventional Commits Specification](https://www.conventionalcommits.org/)
- [GitHub Actions Permissions](https://docs.github.com/en/actions/security-guides/automatic-token-authentication#permissions-for-the-github_token)
- [Managing GitHub Actions Settings](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository)
