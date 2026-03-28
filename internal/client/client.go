package client

import (
	"bytes"
	"context"
	crypt "crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	tls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

const defaultBaseURL = "https://api.ishosting.com"

// Client is the ISHosting API client.
type Client struct {
	BaseURL    string
	APIToken   string
	HTTPClient *http.Client
}

// newUTLSTransport creates an HTTP/2 transport using uTLS with a Chrome TLS fingerprint
// to avoid Cloudflare bot detection that blocks Go's default TLS stack.
func newUTLSTransport() http.RoundTripper {
	dialTLS := func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialer := &net.Dialer{Timeout: 30 * time.Second}
		conn, err := dialer.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}

		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		tlsConn := tls.UClient(conn, &tls.Config{ServerName: host}, tls.HelloChrome_Auto)
		if err := tlsConn.Handshake(); err != nil {
			conn.Close()
			return nil, err
		}
		return tlsConn, nil
	}

	return &http2.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string, _ *crypt.Config) (net.Conn, error) {
			return dialTLS(ctx, network, addr)
		},
	}
}

// NewClient creates a new ISHosting API client.
func NewClient(apiToken, baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		BaseURL:  baseURL,
		APIToken: apiToken,
		HTTPClient: &http.Client{
			Transport: newUTLSTransport(),
			Timeout:   120 * time.Second,
		},
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, query url.Values) ([]byte, error) {
	u := fmt.Sprintf("%s%s", c.BaseURL, path)
	if query != nil {
		u = fmt.Sprintf("%s?%s", u, query.Encode())
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("X-Api-Token", c.APIToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// --- VPS Operations ---

// VPS represents a VPS instance.
type VPS struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Tags     []string `json:"tags"`
	Location struct {
		Name    string `json:"name"`
		Code    string `json:"code"`
		Variant struct {
			Name string `json:"name"`
			Code string `json:"code"`
		} `json:"variant"`
	} `json:"location"`
	Plan struct {
		Name            string  `json:"name"`
		Code            string  `json:"code"`
		Price           float64 `json:"price"`
		Period          string  `json:"period"`
		NextCharge      string  `json:"next_charge"`
		ChargeDeferment string  `json:"charge_deferment"`
		AutoRenew       bool    `json:"auto_renew"`
	} `json:"plan"`
	Platform struct {
		Name   string `json:"name"`
		Code   string `json:"code"`
		Config struct {
			CPU struct {
				Cores int    `json:"cores"`
				Name  string `json:"name"`
			} `json:"cpu"`
			RAM struct {
				Size int    `json:"size"`
				Unit string `json:"unit"`
				Name string `json:"name"`
			} `json:"ram"`
			Drive struct {
				Size int    `json:"size"`
				Unit string `json:"unit"`
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"drive"`
			OS struct {
				Name    string `json:"name"`
				Version string `json:"version"`
				Arch    string `json:"arch"`
			} `json:"os"`
		} `json:"config"`
	} `json:"platform"`
	Network struct {
		PublicIP   string `json:"public_ip"`
		Protocols  struct {
			IPv4 []IPAddress `json:"ipv4"`
			IPv6 []IPAddress `json:"ipv6"`
		} `json:"protocols"`
		Port      int `json:"port"`
		Bandwidth struct {
			Size int    `json:"size"`
			Unit string `json:"unit"`
		} `json:"bandwidth"`
	} `json:"network"`
	Access struct {
		VNC struct {
			Host      string `json:"host"`
			Password  string `json:"password"`
			IsEnabled bool   `json:"is_enabled"`
		} `json:"vnc"`
		SSH struct {
			Users []SSHUser `json:"users"`
			Keys  []string  `json:"keys"`
		} `json:"ssh"`
	} `json:"access"`
	Security struct {
		Backup struct {
			Name string `json:"name"`
			Code string `json:"code"`
		} `json:"backup"`
		DDoS struct {
			Name string `json:"name"`
			Code string `json:"code"`
		} `json:"ddos"`
	} `json:"security"`
	Tools struct {
		Panel struct {
			Name string `json:"name"`
			Code string `json:"code"`
		} `json:"panel"`
		Admin struct {
			Name string `json:"name"`
			Code string `json:"code"`
		} `json:"admin"`
		Virtualization struct {
			Name string `json:"name"`
			Code string `json:"code"`
		} `json:"virtualization"`
	} `json:"tools"`
	Status struct {
		Name    string `json:"name"`
		Code    string `json:"code"`
		Message string `json:"message"`
		State   struct {
			Name    string `json:"name"`
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"state"`
	} `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type IPAddress struct {
	IP      string `json:"ip"`
	Mask    string `json:"mask"`
	Gateway string `json:"gateway"`
	RDNS    string `json:"rdns"`
	IsMain  bool   `json:"is_main"`
}

type SSHUser struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	IsEnabled bool   `json:"is_enabled"`
}

// VPSListResponse wraps the list response.
type VPSListResponse struct {
	Data []VPS `json:"data"`
}

// VPSResponse wraps a single VPS response.
type VPSResponse struct {
	Data VPS `json:"data"`
}

// GetVPS retrieves a VPS by ID.
func (c *Client) GetVPS(ctx context.Context, id string) (*VPS, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/vps/%s", id), nil, nil)
	if err != nil {
		return nil, err
	}

	var resp VPSResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling VPS response: %w", err)
	}

	return &resp.Data, nil
}

// VPSPatchRequest represents the PATCH request body for updating a VPS.
type VPSPatchRequest struct {
	Name *string  `json:"name,omitempty"`
	Tags []string `json:"tags,omitempty"`
	Plan *struct {
		AutoRenew *bool `json:"auto_renew,omitempty"`
	} `json:"plan,omitempty"`
}

// UpdateVPS updates a VPS instance.
func (c *Client) UpdateVPS(ctx context.Context, id string, req VPSPatchRequest) (*VPS, error) {
	respBody, err := c.doRequest(ctx, http.MethodPatch, fmt.Sprintf("/vps/%s", id), req, nil)
	if err != nil {
		return nil, err
	}

	var resp VPSResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling VPS update response: %w", err)
	}

	return &resp.Data, nil
}

// VPSAction performs a status action on a VPS (start, stop, reboot, cancel).
func (c *Client) VPSAction(ctx context.Context, id, action string) error {
	_, err := c.doRequest(ctx, http.MethodPatch, fmt.Sprintf("/vps/%s/status/%s", id, action), nil, nil)
	return err
}

// --- Order Operations ---

// OrderItem represents an item in an order.
type OrderItem struct {
	Action   string `json:"action"`
	Type     string `json:"type"`
	Plan     string `json:"plan"`
	Location struct {
		City string `json:"city"`
	} `json:"location"`
	Quantity  int              `json:"quantity"`
	Options   *OrderOptions    `json:"options,omitempty"`
	Additions []OrderAddition  `json:"additions,omitempty"`
	Comment   string           `json:"comment,omitempty"`
}

type OrderOptions struct {
	VNC *OrderVNC `json:"vnc,omitempty"`
	SSH *OrderSSH `json:"ssh,omitempty"`
}

type OrderVNC struct {
	IsEnabled bool `json:"is_enabled"`
}

type OrderSSH struct {
	IsEnabled bool     `json:"is_enabled"`
	Keys      []string `json:"keys,omitempty"`
}

type OrderAddition struct {
	Code     string          `json:"code,omitempty"`
	Category string          `json:"category"`
	Variant  *OrderVariant   `json:"variant,omitempty"`
}

type OrderVariant struct {
	Lang string `json:"lang,omitempty"`
}

type OrderRequest struct {
	Items  []OrderItem `json:"items"`
	Promos []string    `json:"promos,omitempty"`
}

type InvoiceResponse struct {
	Data struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Items  []struct {
			Identity string `json:"identity"`
			Type     string `json:"type"`
			Plan     string `json:"plan"`
		} `json:"items"`
	} `json:"data"`
}

// CreateOrder creates a new order (provisions a VPS).
func (c *Client) CreateOrder(ctx context.Context, req OrderRequest) (*InvoiceResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/billing/order", req, nil)
	if err != nil {
		return nil, err
	}

	var resp InvoiceResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling order response: %w", err)
	}

	return &resp, nil
}

// WaitForVPSActive polls until the VPS reaches an active state or times out.
func (c *Client) WaitForVPSActive(ctx context.Context, id string, timeout time.Duration) (*VPS, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		vps, err := c.GetVPS(ctx, id)
		if err == nil && vps.Status.State.Code == "ACTIVE" {
			return vps, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}
	return nil, fmt.Errorf("timeout waiting for VPS %s to become active", id)
}

// --- SSH Key Operations ---

type SSHKey struct {
	ID          string `json:"id"`
	Fingerprint string `json:"fingerprint"`
	Title       string `json:"title"`
	Public      string `json:"public"`
}

type SSHKeyResponse struct {
	Data SSHKey `json:"data"`
}

type SSHKeysListResponse struct {
	Data []SSHKey `json:"data"`
}

type SSHKeyCreateRequest struct {
	Title  string `json:"title"`
	Public string `json:"public"`
}

// ListSSHKeys lists all SSH keys.
func (c *Client) ListSSHKeys(ctx context.Context) ([]SSHKey, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/settings/ssh", nil, nil)
	if err != nil {
		return nil, err
	}

	var resp SSHKeysListResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling SSH keys response: %w", err)
	}

	return resp.Data, nil
}

// CreateSSHKey creates a new SSH key.
func (c *Client) CreateSSHKey(ctx context.Context, req SSHKeyCreateRequest) (*SSHKey, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/settings/ssh", req, nil)
	if err != nil {
		return nil, err
	}

	var resp SSHKeyResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling SSH key response: %w", err)
	}

	return &resp.Data, nil
}

// GetSSHKey retrieves a single SSH key by ID (fetches all and filters).
func (c *Client) GetSSHKey(ctx context.Context, id string) (*SSHKey, error) {
	keys, err := c.ListSSHKeys(ctx)
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		if key.ID == id {
			return &key, nil
		}
	}
	return nil, fmt.Errorf("SSH key %s not found", id)
}

// DeleteSSHKey deletes an SSH key by ID.
func (c *Client) DeleteSSHKey(ctx context.Context, id string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/settings/ssh/%s", id), nil, nil)
	return err
}

// --- VPS IP Operations ---

// IPPatchRequest represents the request to update an IP address.
type IPPatchRequest struct {
	IsMain *bool   `json:"is_main,omitempty"`
	RDNS   *string `json:"rdns,omitempty"`
}

// IPResponse represents the API response for IP operations.
type IPResponse struct {
	Data IPAddress `json:"data"`
}

// UpdateVPSIP updates an IP address configuration on a VPS.
func (c *Client) UpdateVPSIP(ctx context.Context, vpsID, protocol, ip string, req IPPatchRequest) (*IPAddress, error) {
	respBody, err := c.doRequest(ctx, http.MethodPatch, fmt.Sprintf("/vps/%s/network/%s/%s", vpsID, protocol, ip), req, nil)
	if err != nil {
		return nil, err
	}

	var resp IPResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling IP response: %w", err)
	}

	return &resp.Data, nil
}

// DeleteVPSIP removes an IP address from a VPS.
func (c *Client) DeleteVPSIP(ctx context.Context, vpsID, protocol, ip string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/vps/%s/network/%s/%s", vpsID, protocol, ip), nil, nil)
	return err
}

// GetVPSIP retrieves a specific IP from a VPS by reading the full VPS and filtering.
func (c *Client) GetVPSIP(ctx context.Context, vpsID, protocol, ip string) (*IPAddress, error) {
	vps, err := c.GetVPS(ctx, vpsID)
	if err != nil {
		return nil, err
	}

	var ips []IPAddress
	switch protocol {
	case "ipv4":
		ips = vps.Network.Protocols.IPv4
	case "ipv6":
		ips = vps.Network.Protocols.IPv6
	default:
		return nil, fmt.Errorf("unknown protocol: %s", protocol)
	}

	for _, addr := range ips {
		if addr.IP == ip {
			return &addr, nil
		}
	}

	return nil, fmt.Errorf("IP %s not found on VPS %s (protocol %s)", ip, vpsID, protocol)
}

// GetVPSIPs returns all IPs for a VPS grouped by protocol.
func (c *Client) GetVPSIPs(ctx context.Context, vpsID string) ([]IPAddress, []IPAddress, string, error) {
	vps, err := c.GetVPS(ctx, vpsID)
	if err != nil {
		return nil, nil, "", err
	}
	return vps.Network.Protocols.IPv4, vps.Network.Protocols.IPv6, vps.Network.PublicIP, nil
}

// --- Plans & Configs ---

type VPSPlan struct {
	Name     string  `json:"name"`
	Code     string  `json:"code"`
	Price    float64 `json:"price"`
	Period   string  `json:"period"`
	Location struct {
		Name    string `json:"name"`
		Code    string `json:"code"`
		Variant struct {
			Name string `json:"name"`
			Code string `json:"code"`
		} `json:"variant"`
	} `json:"location"`
	Platform struct {
		Name   string `json:"name"`
		Code   string `json:"code"`
		Config struct {
			CPU struct {
				Cores int    `json:"cores"`
				Name  string `json:"name"`
			} `json:"cpu"`
			RAM struct {
				Size int    `json:"size"`
				Unit string `json:"unit"`
			} `json:"ram"`
			Drive struct {
				Size int    `json:"size"`
				Unit string `json:"unit"`
				Type string `json:"type"`
			} `json:"drive"`
		} `json:"config"`
	} `json:"platform"`
}

type VPSPlansResponse struct {
	Data []VPSPlan `json:"data"`
}

type VPSPlanResponse struct {
	Data VPSPlan `json:"data"`
}

// ListVPSPlans lists available VPS plans.
func (c *Client) ListVPSPlans(ctx context.Context, locations, platforms []string) ([]VPSPlan, error) {
	query := url.Values{}
	for _, l := range locations {
		query.Add("locations[]", l)
	}
	for _, p := range platforms {
		query.Add("platforms[]", p)
	}

	respBody, err := c.doRequest(ctx, http.MethodGet, "/vps/plans", nil, query)
	if err != nil {
		return nil, err
	}

	var resp VPSPlansResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling VPS plans response: %w", err)
	}

	return resp.Data, nil
}

// VPSConfigsResponse represents available configuration options for a plan.
type VPSConfigsResponse struct {
	Data json.RawMessage `json:"data"`
}

// GetVPSConfigs retrieves available configurations for a plan code.
func (c *Client) GetVPSConfigs(ctx context.Context, planCode string) (json.RawMessage, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/vps/configs/%s", planCode), nil, nil)
	if err != nil {
		return nil, err
	}

	var resp VPSConfigsResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling VPS configs response: %w", err)
	}

	return resp.Data, nil
}
