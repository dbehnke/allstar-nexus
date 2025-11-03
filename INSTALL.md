# Installation Guide

This guide covers different installation methods for Allstar Nexus.

## Table of Contents

- [Debian/Ubuntu Package Installation](#debianubuntu-package-installation)
- [RedHat/CentOS Package Installation](#redhatcentos-package-installation)
- [Manual Installation from Source](#manual-installation-from-source)
- [Binary Archive Installation](#binary-archive-installation)
- [Post-Installation Configuration](#post-installation-configuration)

## Debian/Ubuntu Package Installation

The easiest way to install on Debian-based systems is using the `.deb` package from releases:

```bash
# Download the package from GitHub releases page:
# https://github.com/dbehnke/allstar-nexus/releases/latest
# 
# Replace <version> with the actual version (e.g., v0.10.1)
wget https://github.com/dbehnke/allstar-nexus/releases/download/<version>/allstar-nexus_<version>_linux_amd64.deb

# Install the package
sudo dpkg -i allstar-nexus_*_linux_amd64.deb

# The package automatically:
# - Creates the allstar-nexus system user
# - Installs the binary to /usr/local/bin/allstar-nexus
# - Creates /etc/allstar-nexus/config.yaml (from example)
# - Creates /var/lib/allstar-nexus data directory
# - Installs systemd service (disabled by default)
```

After installation, see [Post-Installation Configuration](#post-installation-configuration).

## RedHat/CentOS Package Installation

For RPM-based distributions:

```bash
# Download the package from GitHub releases page:
# https://github.com/dbehnke/allstar-nexus/releases/latest
#
# Replace <version> with the actual version (e.g., v0.10.1)
wget https://github.com/dbehnke/allstar-nexus/releases/download/<version>/allstar-nexus_<version>_linux_amd64.rpm

# Install the package
sudo rpm -i allstar-nexus_*_linux_amd64.rpm
```

After installation, see [Post-Installation Configuration](#post-installation-configuration).

## Manual Installation from Source

If you prefer to build and install from source:

```bash
# Clone the repository
git clone https://github.com/dbehnke/allstar-nexus.git
cd allstar-nexus

# Build the application (requires Go 1.21+ and Node.js)
make build

# Install (requires root/sudo)
sudo make install

# The install target:
# - Creates the allstar-nexus system user (if needed)
# - Installs the binary to /usr/local/bin/allstar-nexus
# - Creates /etc/allstar-nexus/config.yaml (if not exists)
# - Creates /var/lib/allstar-nexus data directory
# - Installs systemd service file
```

After installation, see [Post-Installation Configuration](#post-installation-configuration).

## Binary Archive Installation

Download and extract a pre-built archive:

```bash
# Download for your platform from:
# https://github.com/dbehnke/allstar-nexus/releases/latest
#
# Replace <version> with the actual version (e.g., v0.10.1)
wget https://github.com/dbehnke/allstar-nexus/releases/download/<version>/allstar-nexus_<version>_Linux_x86_64.tar.gz

# Extract
tar xzf allstar-nexus_*_Linux_x86_64.tar.gz

# Move binary to your preferred location
sudo mv allstar-nexus /usr/local/bin/

# Create config directory and copy example config
sudo mkdir -p /etc/allstar-nexus
sudo cp config.yaml.example /etc/allstar-nexus/config.yaml

# Optionally install the systemd service
sudo cp allstar-nexus.service /etc/systemd/system/
sudo systemctl daemon-reload
```

## Post-Installation Configuration

### 1. Edit the Configuration File

```bash
sudo nano /etc/allstar-nexus/config.yaml
```

**Important settings to configure:**
- `ami_host`, `ami_port`, `ami_username`, `ami_password` - Your AllStarLink Asterisk Manager Interface credentials
- `nodes` - Your AllStarLink node number(s)
- `jwt_secret` - Change this to a strong random string for production
- `port` - Web interface port (default: 8080)

See `config.yaml.example` for all available options and documentation.

### 2. Enable and Start the Service

```bash
# Reload systemd configuration
sudo systemctl daemon-reload

# Enable service to start on boot
sudo systemctl enable allstar-nexus

# Start the service
sudo systemctl start allstar-nexus

# Check status
sudo systemctl status allstar-nexus
```

### 3. View Logs

```bash
# Follow logs in real-time
sudo journalctl -u allstar-nexus -f

# View recent logs
sudo journalctl -u allstar-nexus -n 100
```

### 4. Access the Web Interface

Open your browser to:
```
http://your-server-ip:8080
```

The first user you register will become the superadmin.

## Systemd Service Details

The systemd service is configured with:
- **User/Group**: Runs as dedicated `allstar-nexus` system user
- **Working Directory**: `/var/lib/allstar-nexus` (for database and data files)
- **Restart Policy**: Always restart with 10-second delay on failure
- **Dependencies**: Waits for network to be online before starting
- **Security**: Hardened with `NoNewPrivileges`, `PrivateTmp`, `ProtectSystem=strict`, `ProtectHome`

## Uninstallation

### Package Installation

```bash
# Debian/Ubuntu
sudo dpkg -r allstar-nexus

# RedHat/CentOS
sudo rpm -e allstar-nexus
```

Configuration and data files are preserved in `/etc/allstar-nexus` and `/var/lib/allstar-nexus`.

### Manual Installation

```bash
cd /path/to/allstar-nexus
sudo make uninstall
```

### Removing Configuration and Data

If you want to completely remove all traces:

```bash
sudo rm -rf /etc/allstar-nexus
sudo rm -rf /var/lib/allstar-nexus
sudo userdel allstar-nexus
```

## Troubleshooting

### Service won't start

Check logs:
```bash
sudo journalctl -u allstar-nexus -n 50
```

Common issues:
- Invalid configuration file (check YAML syntax)
- Cannot connect to AMI (check credentials and network)
- Port already in use (check `port` setting)
- Database permissions (ensure `/var/lib/allstar-nexus` is writable by `allstar-nexus` user)

### Permission errors

Ensure the data directory has correct ownership:
```bash
sudo chown -R allstar-nexus:allstar-nexus /var/lib/allstar-nexus
```

### Database location

By default, the database is stored at `/var/lib/allstar-nexus/data/allstar.db`. You can change this in the config file with the `db_path` setting (use absolute paths or paths relative to the working directory `/var/lib/allstar-nexus`).

## Upgrading

### Package Installation

Download and install the new package version:
```bash
# Debian/Ubuntu (replace <version> with actual version like v0.10.2)
sudo dpkg -i allstar-nexus_<version>_linux_amd64.deb

# RedHat/CentOS
sudo rpm -U allstar-nexus_<version>_linux_amd64.rpm
```

The upgrade process preserves your configuration file.

### Manual Installation

```bash
cd /path/to/allstar-nexus
git pull
make build
sudo make install
sudo systemctl restart allstar-nexus
```

## Building Packages

To build Debian/RPM packages yourself using GoReleaser:

```bash
# Install GoReleaser (https://goreleaser.com/install/)

# Build packages (requires building frontend first)
make build-dashboard
goreleaser release --snapshot --clean

# Packages will be in dist/ directory
```
