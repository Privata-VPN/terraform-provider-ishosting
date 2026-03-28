package datasources

import (
	"context"
	"fmt"

	"terraform-provider-ishosting/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &VPSIPsDataSource{}
	_ datasource.DataSourceWithConfigure = &VPSIPsDataSource{}
)

type VPSIPsDataSource struct {
	client *client.Client
}

type VPSIPsDataSourceModel struct {
	VPSID    types.String `tfsdk:"vps_id"`
	PublicIP types.String `tfsdk:"public_ip"`
	IPv4     []IPModel    `tfsdk:"ipv4"`
	IPv6     []IPModel    `tfsdk:"ipv6"`
}

type IPModel struct {
	Address types.String `tfsdk:"address"`
	Mask    types.String `tfsdk:"mask"`
	Gateway types.String `tfsdk:"gateway"`
	RDNS    types.String `tfsdk:"rdns"`
	IsMain  types.Bool   `tfsdk:"is_main"`
}

func NewVPSIPsDataSource() datasource.DataSource {
	return &VPSIPsDataSource{}
}

func (d *VPSIPsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vps_ips"
}

func (d *VPSIPsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	ipAttributes := map[string]schema.Attribute{
		"address": schema.StringAttribute{
			Description: "IP address.",
			Computed:    true,
		},
		"mask": schema.StringAttribute{
			Description: "Subnet mask.",
			Computed:    true,
		},
		"gateway": schema.StringAttribute{
			Description: "Gateway address.",
			Computed:    true,
		},
		"rdns": schema.StringAttribute{
			Description: "Reverse DNS record.",
			Computed:    true,
		},
		"is_main": schema.BoolAttribute{
			Description: "Whether this is the main IP.",
			Computed:    true,
		},
	}

	resp.Schema = schema.Schema{
		Description: "Retrieves all IP addresses assigned to an ISHosting VPS instance.",
		Attributes: map[string]schema.Attribute{
			"vps_id": schema.StringAttribute{
				Description: "The VPS instance ID.",
				Required:    true,
			},
			"public_ip": schema.StringAttribute{
				Description: "The primary public IP address of the VPS.",
				Computed:    true,
			},
			"ipv4": schema.ListNestedAttribute{
				Description: "List of IPv4 addresses.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: ipAttributes,
				},
			},
			"ipv6": schema.ListNestedAttribute{
				Description: "List of IPv6 addresses.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: ipAttributes,
				},
			},
		},
	}
}

func (d *VPSIPsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *VPSIPsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state VPSIPsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpsID := state.VPSID.ValueString()
	ipv4s, ipv6s, publicIP, err := d.client.GetVPSIPs(ctx, vpsID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VPS IPs",
			"Could not read IPs for VPS "+vpsID+": "+err.Error(),
		)
		return
	}

	state.PublicIP = types.StringValue(publicIP)

	state.IPv4 = make([]IPModel, len(ipv4s))
	for i, ip := range ipv4s {
		state.IPv4[i] = IPModel{
			Address: types.StringValue(ip.Address),
			Mask:    types.StringValue(ip.Mask),
			Gateway: types.StringValue(ip.Gateway),
			RDNS:    types.StringValue(ip.RDNS),
			IsMain:  types.BoolValue(ip.IsMain),
		}
	}

	state.IPv6 = make([]IPModel, len(ipv6s))
	for i, ip := range ipv6s {
		state.IPv6[i] = IPModel{
			Address: types.StringValue(ip.Address),
			Mask:    types.StringValue(ip.Mask),
			Gateway: types.StringValue(ip.Gateway),
			RDNS:    types.StringValue(ip.RDNS),
			IsMain:  types.BoolValue(ip.IsMain),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
