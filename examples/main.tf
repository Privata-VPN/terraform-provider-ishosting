terraform {
  required_providers {
    ishosting = {
      source = "registry.terraform.io/ishosting/ishosting"
    }
  }
}

# Configure the ISHosting provider.
# Set the ISHOSTING_API_TOKEN environment variable or use the api_token attribute.
provider "ishosting" {
  # api_token = "your-api-token-here"
}

# ─── Data Sources ───────────────────────────────────────────────

# Look up available VPS plans
data "ishosting_vps_plans" "all" {}

# Look up available configs for a specific plan
data "ishosting_vps_configs" "plan_configs" {
  plan_code = "vps-kvm-lin-1-ber"
}

# ─── SSH Key ────────────────────────────────────────────────────

# Create an SSH key for VPS access
resource "ishosting_ssh_key" "my_key" {
  title      = "my-terraform-key"
  public_key = file("~/.ssh/id_rsa.pub")
}

# ─── VPS Instance ──────────────────────────────────────────────

# Provision a VPS instance
resource "ishosting_vps" "web_server" {
  plan     = "vps-kvm-lin-1-ber-1m"
  location = "ber"
  name     = "web-server-01"
  tags     = ["web", "production"]

  # OS selection (use ishosting_vps_configs data source to find codes)
  os_category = "os_linux_ubuntu"
  os_code     = "ubuntu_22_04_64"

  # Access settings
  vnc_enabled = false
  ssh_enabled = true
  ssh_keys    = [ishosting_ssh_key.my_key.id]

  auto_renew = true
}

# ─── VPS IP Management ─────────────────────────────────────────

# Read all IPs assigned to the VPS
data "ishosting_vps_ips" "web_server_ips" {
  vps_id = ishosting_vps.web_server.id
}

# Manage the primary IPv4 - set reverse DNS
resource "ishosting_vps_ip" "web_server_main" {
  vps_id   = ishosting_vps.web_server.id
  protocol = "ipv4"
  address  = ishosting_vps.web_server.public_ip
  rdns     = "web-server-01.example.com"
  is_main  = true
}

# ─── Outputs ───────────────────────────────────────────────────

output "vps_id" {
  value = ishosting_vps.web_server.id
}

output "vps_public_ip" {
  value = ishosting_vps.web_server.public_ip
}

output "vps_status" {
  value = ishosting_vps.web_server.status
}

output "vps_ipv4_addresses" {
  value = data.ishosting_vps_ips.web_server_ips.ipv4
}

output "vps_ipv6_addresses" {
  value = data.ishosting_vps_ips.web_server_ips.ipv6
}

output "available_plans" {
  value = [for p in data.ishosting_vps_plans.all.plans : {
    code     = p.code
    name     = p.name
    price    = p.price
    location = p.city_name
    cpu      = p.cpu_cores
    ram      = "${p.ram_size} ${p.ram_unit}"
    drive    = "${p.drive_size} ${p.drive_unit} ${p.drive_type}"
  }]
}
