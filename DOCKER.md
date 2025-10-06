# Docker Deployment Guide

This guide explains how to run Allstar Nexus using Docker.

## Quick Start

### 1. Build the Image

```bash
docker-compose build
```

### 2. Configure

Create a `data/config.yaml` file with your settings (or use environment variables in docker-compose.yml):

```yaml
# data/config.yaml
title: "My Allstar Hub"
subtitle: "K8FBI - Michigan"

nodes: [594950]

ami_host: 192.168.1.100  # Your Asterisk server IP
ami_port: 5038
ami_username: admin
ami_password: your-password

jwt_secret: change-me-to-random-string
```

### 3. Run

```bash
docker-compose up -d
```

The application will be available at http://localhost:8080

## Configuration

### Option 1: Config File (Recommended)

Create `data/config.yaml` and mount it as a volume. The application automatically searches for config files in:
- `/app/data/config.yaml` (Docker default)
- `./config.yaml`
- `$HOME/.allstar-nexus/config.yaml`
- `/etc/allstar-nexus/config.yaml`

### Option 2: Environment Variables

All settings can be configured via environment variables in `docker-compose.yml`:

```yaml
environment:
  - AMI_HOST=192.168.1.100
  - AMI_PORT=5038
  - AMI_USERNAME=admin
  - AMI_PASSWORD=your-password
  - NODES=[594950,43732]
  - JWT_SECRET=change-me
```

See `config.yaml.example` for all available options.

## Volumes

The `data/` directory is mounted as a volume for persistence:
- **Database**: `/app/data/allstar.db` - User accounts, link statistics
- **AstDB Cache**: `/app/data/astdb.txt` - Node lookup database
- **Config**: `/app/data/config.yaml` - Optional config file

## Networking

### Host Network Mode (for local Asterisk)

If your Asterisk server is on the same host:

```yaml
services:
  allstar-nexus:
    network_mode: host
    environment:
      - AMI_HOST=127.0.0.1
```

### Bridge Mode (for remote Asterisk)

Default mode - connects to remote Asterisk server via IP address.

## Health Checks

The container includes a health check that pings `/api/health` every 30 seconds.

View health status:
```bash
docker-compose ps
```

## Logs

View logs:
```bash
docker-compose logs -f
```

## Updates

Pull latest image and restart:
```bash
docker-compose pull
docker-compose up -d
```

Or rebuild from source:
```bash
docker-compose build --no-cache
docker-compose up -d
```

## Security Notes

1. **Change default secrets**: Set `JWT_SECRET` to a random string
2. **Use strong AMI password**: Configure Asterisk with a strong password
3. **Firewall**: Limit port 8080 access if needed
4. **User**: Container runs as UID 1000 (non-root)

## Troubleshooting

### Permission Issues

If you get permission errors:
```bash
sudo chown -R 1000:1000 data/
```

### AMI Connection Issues

Check connectivity from container:
```bash
docker-compose exec allstar-nexus wget -O- http://localhost:8080/api/health
```

### View Container Environment

```bash
docker-compose exec allstar-nexus env
```

## Advanced Configuration

### Custom Port

```yaml
ports:
  - "3000:8080"  # Host:Container
environment:
  - PORT=8080    # Internal port (keep as 8080)
```

### Resource Limits

```yaml
deploy:
  resources:
    limits:
      cpus: '1'
      memory: 512M
```

### Named Volume

Instead of bind mount:
```yaml
volumes:
  - allstar-data:/app/data

volumes:
  allstar-data:
```
