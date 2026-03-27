# Terraform Provider for ISHosting

Manage [ISHosting](https://ishosting.com) VPS instances with Terraform.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- Go 1.24+ (only if building from source)
- An ISHosting API token — generate one in the ISHosting control panel

---

## Installation (GitHub Release)

This provider is distributed as a binary via [GitHub Releases](https://github.com/Privata-VPN/ishosting-terraform-provider/releases).

### Step 1 — Download the binary

Pick the archive for your platform from the latest release:

| Platform | File |
|----------|------|
| Linux amd64 | `terraform-provider-ishosting_linux_amd64.tar.gz` |
| Linux arm64 | `terraform-provider-ishosting_linux_arm64.tar.gz` |
| macOS amd64 | `terraform-provider-ishosting_darwin_amd64.tar.gz` |
| macOS arm64 (Apple Silicon) | `terraform-provider-ishosting_darwin_arm64.tar.gz` |
| Windows amd64 | `terraform-provider-ishosting_windows_amd64.zip` |

```bash
# Example: macOS Apple Silicon
VERSION=v0.1.5   # replace with the latest release tag
curl -L https://github.com/Privata-VPN/ishosting-terraform-provider/releases/download/${VERSION}/terraform-provider-ishosting_darwin_arm64.tar.gz \
  | tar xz
```

### Step 2 — Place the binary in the local plugins directory

Terraform looks for local providers in:

```
~/.terraform.d/plugins/<hostname>/<namespace>/<type>/<version>/<os_arch>/
```

```bash
# Set your platform and version
HOSTNAME=github.com
NAMESPACE=privata-vpn
TYPE=ishosting
VERSION=0.1.5          # without the "v" prefix
OS_ARCH=darwin_arm64   # match your platform from the table above

PLUGIN_DIR=~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${TYPE}/${VERSION}/${OS_ARCH}

mkdir -p "$PLUGIN_DIR"
mv terraform-provider-ishosting "$PLUGIN_DIR/terraform-provider-${TYPE}_v${VERSION}"
chmod +x "$PLUGIN_DIR/terraform-provider-${TYPE}_v${VERSION}"
```

### Step 3 — Configure your Terraform project

```hcl
# versions.tf
terraform {
  required_providers {
    ishosting = {
      source  = "github.com/privata-vpn/ishosting"
      version = "~> 0.1"
    }
  }
}
```

Then run:

```bash
terraform init
```

---

## Alternative: dev_overrides (no versioned directory needed)

If you just want to point Terraform at the binary without the full directory structure, add this to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "github.com/privata-vpn/ishosting" = "/path/to/ishosting-terraform-provider"
  }
  direct {}
}
```

Place the built binary in that directory and **skip `terraform init`** — Terraform will use the binary directly.

---

## Provider Configuration

```hcl
provider "ishosting" {
  api_token = "your-api-token"   # or set ISHOSTING_API_TOKEN env var
}
```

| Argument | Required | Description |
|----------|----------|-------------|
| `api_token` | yes | ISHosting API token. Can also be set via `ISHOSTING_API_TOKEN`. |
| `base_url` | no | API base URL. Defaults to `https://api.ishosting.com`. |

**Recommended:** use an environment variable instead of hardcoding the token:

```bash
export ISHOSTING_API_TOKEN="your-api-token"
```

---

## Resources & Data Sources

### Resources

| Resource | Description |
|----------|-------------|
| `ishosting_vps` | Provision and manage a VPS instance |
| `ishosting_ssh_key` | Manage SSH keys |
| `ishosting_vps_ip` | Manage IP address settings (RDNS, main IP) |

### Data Sources

| Data Source | Description |
|-------------|-------------|
| `ishosting_vps_plans` | List available VPS plans |
| `ishosting_vps_configs` | Get configuration options for a plan |
| `ishosting_vps_ips` | List all IPs assigned to a VPS |

---

## Usage Example

```hcl
terraform {
  required_providers {
    ishosting = {
      source  = "github.com/privata-vpn/ishosting"
      version = "~> 0.1"
    }
  }
}

provider "ishosting" {
  # api_token read from ISHOSTING_API_TOKEN env var
}

# Upload your SSH public key
resource "ishosting_ssh_key" "default" {
  title      = "my-key"
  public_key = file("~/.ssh/id_rsa.pub")
}

# Browse available plans
data "ishosting_vps_plans" "all" {}

# Provision a VPS
resource "ishosting_vps" "web" {
  plan      = "vps-kvm-lin-1-ber-1m"
  location  = "ber"
  name      = "web-01"
  tags      = ["web", "production"]

  os_category = "os_linux_ubuntu"
  os_code     = "ubuntu_22_04_64"

  ssh_enabled = true
  ssh_keys    = [ishosting_ssh_key.default.id]

  auto_renew = true
}

# Set reverse DNS on the main IP
resource "ishosting_vps_ip" "main" {
  vps_id   = ishosting_vps.web.id
  protocol = "ipv4"
  address  = ishosting_vps.web.public_ip
  rdns     = "web-01.example.com"
  is_main  = true
}

# Read all IPs
data "ishosting_vps_ips" "web" {
  vps_id = ishosting_vps.web.id
}

output "public_ip" {
  value = ishosting_vps.web.public_ip
}

output "all_ipv4" {
  value = data.ishosting_vps_ips.web.ipv4[*].address
}
```

### Import an existing IP

```bash
terraform import ishosting_vps_ip.main <vps_id>/ipv4/<ip_address>
```

---

## Building from Source

```bash
git clone https://github.com/Privata-VPN/ishosting-terraform-provider.git
cd ishosting-terraform-provider
make install   # builds and installs to ~/.terraform.d/plugins/
```

---

## License

MIT
