package resources

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-ishosting/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &VPSIPResource{}
	_ resource.ResourceWithConfigure   = &VPSIPResource{}
	_ resource.ResourceWithImportState = &VPSIPResource{}
)

type VPSIPResource struct {
	client *client.Client
}

type VPSIPResourceModel struct {
	ID       types.String `tfsdk:"id"`
	VPSID    types.String `tfsdk:"vps_id"`
	Protocol types.String `tfsdk:"protocol"`
	Address  types.String `tfsdk:"address"`
	RDNS     types.String `tfsdk:"rdns"`
	IsMain   types.Bool   `tfsdk:"is_main"`
	Mask     types.String `tfsdk:"mask"`
	Gateway  types.String `tfsdk:"gateway"`
}

func NewVPSIPResource() resource.Resource {
	return &VPSIPResource{}
}

func (r *VPSIPResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vps_ip"
}

func (r *VPSIPResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an IP address on an ISHosting VPS instance. " +
			"This resource manages the configuration (RDNS, is_main) of an existing IP. " +
			"IPs are allocated when the VPS is ordered. " +
			"Destroying this resource will remove the IP from the VPS.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Resource ID in the format vps_id/protocol/address.",
				Computed:    true,
			},
			"vps_id": schema.StringAttribute{
				Description: "The VPS instance ID this IP belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"protocol": schema.StringAttribute{
				Description: "IP protocol: ipv4 or ipv6.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("ipv4", "ipv6"),
				},
			},
			"address": schema.StringAttribute{
				Description: "The IP address.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rdns": schema.StringAttribute{
				Description: "Reverse DNS record for this IP.",
				Optional:    true,
				Computed:    true,
			},
			"is_main": schema.BoolAttribute{
				Description: "Whether this is the main IP for the VPS.",
				Optional:    true,
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
		},
	}
}

func (r *VPSIPResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *VPSIPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan VPSIPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpsID := plan.VPSID.ValueString()
	protocol := plan.Protocol.ValueString()
	address := plan.Address.ValueString()

	// Build the patch request with any provided values
	patchReq := client.IPPatchRequest{}
	needsPatch := false

	if !plan.RDNS.IsNull() && !plan.RDNS.IsUnknown() {
		rdns := plan.RDNS.ValueString()
		patchReq.RDNS = &rdns
		needsPatch = true
	}
	if !plan.IsMain.IsNull() && !plan.IsMain.IsUnknown() {
		isMain := plan.IsMain.ValueBool()
		patchReq.IsMain = &isMain
		needsPatch = true
	}

	if needsPatch {
		_, err := r.client.UpdateVPSIP(ctx, vpsID, protocol, address, patchReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Configuring VPS IP",
				fmt.Sprintf("Could not configure IP %s on VPS %s: %s", address, vpsID, err.Error()),
			)
			return
		}
	}

	// Read back the current state
	ipAddr, err := r.client.GetVPSIP(ctx, vpsID, protocol, address)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VPS IP",
			fmt.Sprintf("Could not read IP %s on VPS %s: %s", address, vpsID, err.Error()),
		)
		return
	}

	r.mapIPToModel(ipAddr, vpsID, protocol, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *VPSIPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state VPSIPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpsID := state.VPSID.ValueString()
	protocol := state.Protocol.ValueString()
	address := state.Address.ValueString()

	ipAddr, err := r.client.GetVPSIP(ctx, vpsID, protocol, address)
	if err != nil {
		// If the IP is gone, remove from state
		if strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading VPS IP",
			fmt.Sprintf("Could not read IP %s on VPS %s: %s", address, vpsID, err.Error()),
		)
		return
	}

	r.mapIPToModel(ipAddr, vpsID, protocol, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VPSIPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan VPSIPResourceModel
	var state VPSIPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpsID := state.VPSID.ValueString()
	protocol := state.Protocol.ValueString()
	address := state.Address.ValueString()

	patchReq := client.IPPatchRequest{}

	if !plan.RDNS.Equal(state.RDNS) {
		rdns := plan.RDNS.ValueString()
		patchReq.RDNS = &rdns
	}
	if !plan.IsMain.Equal(state.IsMain) {
		isMain := plan.IsMain.ValueBool()
		patchReq.IsMain = &isMain
	}

	_, err := r.client.UpdateVPSIP(ctx, vpsID, protocol, address, patchReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating VPS IP",
			fmt.Sprintf("Could not update IP %s on VPS %s: %s", address, vpsID, err.Error()),
		)
		return
	}

	// Read back
	ipAddr, err := r.client.GetVPSIP(ctx, vpsID, protocol, address)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VPS IP After Update",
			fmt.Sprintf("Could not read IP %s on VPS %s: %s", address, vpsID, err.Error()),
		)
		return
	}

	r.mapIPToModel(ipAddr, vpsID, protocol, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *VPSIPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state VPSIPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteVPSIP(ctx, state.VPSID.ValueString(), state.Protocol.ValueString(), state.Address.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting VPS IP",
			fmt.Sprintf("Could not delete IP %s from VPS %s: %s", state.Address.ValueString(), state.VPSID.ValueString(), err.Error()),
		)
		return
	}
}

func (r *VPSIPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: vps_id/protocol/address
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in the format: vps_id/protocol/address (e.g., abc123/ipv4/1.2.3.4)",
		)
		return
	}

	vpsID := parts[0]
	protocol := parts[1]
	address := parts[2]

	ipAddr, err := r.client.GetVPSIP(ctx, vpsID, protocol, address)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing VPS IP",
			fmt.Sprintf("Could not read IP %s on VPS %s: %s", address, vpsID, err.Error()),
		)
		return
	}

	var state VPSIPResourceModel
	r.mapIPToModel(ipAddr, vpsID, protocol, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *VPSIPResource) mapIPToModel(ip *client.IPAddress, vpsID, protocol string, model *VPSIPResourceModel) {
	model.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", vpsID, protocol, ip.IP))
	model.VPSID = types.StringValue(vpsID)
	model.Protocol = types.StringValue(protocol)
	model.Address = types.StringValue(ip.IP)
	model.RDNS = types.StringValue(ip.RDNS)
	model.IsMain = types.BoolValue(ip.IsMain)
	model.Mask = types.StringValue(ip.Mask)
	model.Gateway = types.StringValue(ip.Gateway)
}
