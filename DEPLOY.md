# Deployment Guide

## One-time setup

### 1. Google OAuth credentials

1. Go to [console.cloud.google.com/apis/credentials](https://console.cloud.google.com/apis/credentials)
2. Create a new **OAuth 2.0 Client ID** (Web application)
3. Add authorised redirect URI: `https://api.recipeasy.yourdomain.com/v1/auth/google/callback`
4. Note the **Client ID** and **Client Secret**

### 2. Gemini API key

1. Go to [aistudio.google.com/app/apikey](https://aistudio.google.com/app/apikey)
2. Create a new API key
3. Note it for the `.env` file below

### 3. DNS

Point two A records at your Droplet's IP address:

```
recipeasy.yourdomain.com      A  <droplet-ip>
api.recipeasy.yourdomain.com  A  <droplet-ip>
```

### 4. Droplet setup

SSH into your Droplet and run:

```bash
apt update && apt install -y docker.io docker-compose-plugin
systemctl enable --now docker

# Create app directory
mkdir -p /opt/recipeasy/infra/images
cd /opt/recipeasy

# Clone the repo
git clone https://github.com/yourusername/recipeasy.git .
```

### 5. Create the .env file

```bash
cp infra/.env.example infra/.env
nano infra/.env
```

Fill in every value — see [infra/.env.example](infra/.env.example) for the full list. Key ones:

| Variable | Value |
|---|---|
| `GITHUB_OWNER` | Your GitHub username |
| `POSTGRES_PASSWORD` | A strong random password |
| `JWT_SECRET` | A random 64-character string (`openssl rand -hex 32`) |
| `GOOGLE_CLIENT_ID` | From step 1 |
| `GOOGLE_CLIENT_SECRET` | From step 1 |
| `GOOGLE_REDIRECT_URL` | `https://api.recipeasy.yourdomain.com/v1/auth/google/callback` |
| `ALLOWED_EMAILS` | `you@gmail.com,wife@gmail.com` |
| `GEMINI_API_KEY` | From step 2 |
| `FRONTEND_URL` | `https://recipeasy.yourdomain.com` |

### 6. Update the Caddyfile

Replace `recipeasy.yourdomain.com` with your actual domain in [infra/Caddyfile](infra/Caddyfile).

### 7. Add PWA icons

Add two PNG icons to `web/public/`:
- `icon-192.png` — 192×192px
- `icon-512.png` — 512×512px

### 8. GitHub Actions secrets

In your GitHub repo → Settings → Secrets → Actions, add:

| Secret | Value |
|---|---|
| `DROPLET_HOST` | Your Droplet's IP or hostname |
| `DROPLET_USER` | `root` (or your SSH user) |
| `DROPLET_SSH_KEY` | Contents of your private SSH key |
| `VITE_API_URL` | `https://api.recipeasy.yourdomain.com/v1` |

### 9. First deploy

On the Droplet:

```bash
cd /opt/recipeasy/infra

# Pull and start everything
docker compose up -d

# Check all three containers are running
docker compose ps
```

Caddy will automatically obtain TLS certificates on first startup. Check it worked:

```bash
docker compose logs caddy
```

### 10. Build and deploy the web app

On your local machine:

```bash
cd web
VITE_API_URL=https://api.recipeasy.yourdomain.com/v1 npm run build

# Copy the build to the Droplet
scp -r dist/* root@<droplet-ip>:/opt/recipeasy/infra/web/
```

After this, all future deploys happen automatically via `git push` to `main`.

---

## Ongoing deploys

Push to `main` — GitHub Actions will:

1. Run Go tests
2. Build and push the Docker image to GitHub Container Registry
3. SSH into the Droplet and pull the new image
4. Build the React PWA and copy it to the Droplet

---

## Useful commands on the Droplet

```bash
# View logs
docker compose -f /opt/recipeasy/infra/docker-compose.yml logs -f api

# Restart the API
docker compose -f /opt/recipeasy/infra/docker-compose.yml restart api

# Connect to the database
docker exec -it recipeasy-infra-postgres-1 psql -U recipeasy recipeasy

# Manual backup
docker exec recipeasy-infra-postgres-1 pg_dump -U recipeasy recipeasy \
  | gzip > /opt/recipeasy/backup-$(date +%Y%m%d).sql.gz
```

---

## Set up daily database backups

On the Droplet:

```bash
cat > /etc/cron.daily/recipeasy-backup << 'EOF'
#!/bin/bash
docker exec recipeasy-infra-postgres-1 pg_dump -U recipeasy recipeasy \
  | gzip > /opt/recipeasy/backups/$(date +%Y%m%d).sql.gz
find /opt/recipeasy/backups -mtime +30 -delete
EOF

chmod +x /etc/cron.daily/recipeasy-backup
mkdir -p /opt/recipeasy/backups
```
