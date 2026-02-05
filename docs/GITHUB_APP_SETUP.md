# GitHub App Setup Guide

This guide explains how to set up AgntPR with GitHub App authentication.

## For App Owners (Building/Distributing AgntPR)

If you're building and distributing AgntPR, you need to create the GitHub
App and bake credentials into the container.

### Step 1: Create GitHub App

1. Go to https://github.com/settings/apps
2. Click **"New GitHub App"**
3. Configure:
   - **Name**: `AgntPR` (or your preferred name)
   - **Homepage URL**: `https://github.com/joaomdsg/agntpr`
   - **Webhook**: Uncheck "Active" (we poll, don't use webhooks)

4. **Permissions** (Repository):
   - Contents: **Read & Write**
   - Issues: **Read & Write**
   - Pull requests: **Read & Write**
   - Metadata: **Read** (automatic)

5. **Where can this app be installed?**: "Any account"
6. Click **"Create GitHub App"**

### Step 2: Generate Private Key

1. After creation, scroll to **"Private keys"**
2. Click **"Generate a private key"**
3. Save the `.pem` file (e.g., `agntpr-private-key.pem`)
4. Note the **App ID** (e.g., `123456`)

### Step 3: Configure Build Credentials

Create `internal/auth/credentials.txt`:

```
123456
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
(paste your private key here)
...
-----END RSA PRIVATE KEY-----
```

Format:
- Line 1: App ID
- Lines 2+: Private key (including BEGIN/END lines)

### Step 4: Build Container

```bash
# Credentials are embedded at build time
docker build -t agntpr:latest .

# Or with make
make docker
```

**Security**: The private key is embedded in the binary using `go:embed`.
Users cannot extract or override it at runtime.

### Step 5: Publish

```bash
docker push yourdockerhub/agntpr:latest
```

Users will install YOUR GitHub App and run YOUR container with just their
installation ID.

---

## For Users (Installing AgntPR)

### Step 1: Install the AgntPR App

1. Go to the App installation page (provided by the app owner)
2. Click **"Install"**
3. Choose repositories to grant access
4. After install, note the **Installation ID** from the URL:
   ```
   https://github.com/settings/installations/12345678
                                              ^^^^^^^^
                                              This is your Installation ID
   ```

### Step 2: Run AgntPR

```bash
docker run -d \
  -e GITHUB_APP_INSTALLATION_ID=12345678 \
  -e TARGET_REPO=youruser/yourrepo \
  agntpr:latest
```

Or with Docker Compose:

```yaml
services:
  agntpr:
    image: agntpr:latest
    environment:
      GITHUB_APP_INSTALLATION_ID: 12345678
      TARGET_REPO: youruser/yourrepo
      POLL_INTERVAL: 60
    restart: unless-stopped
```

### Step 3: Use It

1. Create an issue in your repo
2. Mention `@AgntPR[bot]` in the issue
3. The bot will respond with a plan
4. Approve the plan in comments
5. Bot implements and creates a PR

---

## Environment Variables (Users)

**Required:**
- `GITHUB_APP_INSTALLATION_ID` - Your installation ID (from install URL)
- `TARGET_REPO` - Repository to watch (format: `owner/repo`)

**Optional:**
- `POLL_INTERVAL` - Polling interval in seconds (default: `60`)
- `DEBUG` - Enable debug logging (`true` or `false`)
- `WORK_DIR` - Working directory for repos (default: `/work`)

**Not Configurable:**
- App ID and private key are embedded at build time
- Cannot be overridden at runtime

---

## Security Notes

**For App Owners:**
- Keep your private key secure
- Never commit `credentials.txt` to git
- Rotate keys periodically in GitHub App settings
- Users must trust your container (like any software)

**For Users:**
- Installation ID is not sensitive (it's in the URL)
- Review app permissions before installing
- You can uninstall the app anytime to revoke access
- Bot only has access to repositories you approve

---

## Troubleshooting

**"No embedded credentials found"**
- The container was built without credentials
- App owner needs to rebuild with `credentials.txt`

**"GITHUB_APP_INSTALLATION_ID is required"**
- You forgot to set the installation ID
- Get it from the installation URL

**"Token expired, GitHub API call needed"** with cached token
- Token refresh failed
- Check GitHub App is still installed
- Verify the app hasn't been suspended

**Bot not responding to mentions**
- Check the bot username matches your mentions
- Verify `TARGET_REPO` is correct
- Check logs with `docker logs <container>`

---

## Legacy Token Auth

AgntPR also supports personal access tokens (legacy mode):

```bash
docker run -d \
  -e GITHUB_TOKEN=ghp_your_token \
  -e TARGET_REPO=youruser/yourrepo \
  agntpr:latest
```

This mode doesn't require creating a GitHub App, but the bot will act as
your personal account instead of showing as `AgntPR[bot]`.
