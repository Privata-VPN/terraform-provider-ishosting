package resources

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"terraform-provider-ishosting/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource              = &VPSResource{}
	_ resource.ResourceWithConfigure = &VPSResource{}
)

type VPSResource struct {
	client *client.Client
}

type VPSResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Tags      types.List   `tfsdk:"tags"`
	Plan      types.String `tfsdk:"plan"`
	Location  types.String `tfsdk:"location"`
	AutoRenew types.Bool   `tfsdk:"auto_renew"`

	// Order options
	OSCategory types.String `tfsdk:"os_category"`
	OSCode     types.String `tfsdk:"os_code"`
	VNCEnabled types.Bool   `tfsdk:"vnc_enabled"`
	SSHEnabled types.Bool   `tfsdk:"ssh_enabled"`
	SSHKeys    types.List   `tfsdk:"ssh_keys"`
	Quantity   types.Int64  `tfsdk:"quantity"`
	Comment    types.String `tfsdk:"comment"`
	Promos     types.List   `tfsdk:"promos"`

	// Additions
	Additions types.List `tfsdk:"additions"`

	// Internal tracking
	InvoiceID types.String `tfsdk:"invoice_id"`

	// Computed
	PublicIP       types.String `tfsdk:"public_ip"`
	Status         types.String `tfsdk:"status"`
	State          types.String `tfsdk:"state"`
	PlatformName   types.String `tfsdk:"platform_name"`
	CPUCores       types.Int64  `tfsdk:"cpu_cores"`
	RAMSize        types.Int64  `tfsdk:"ram_size"`
	RAMUnit        types.String `tfsdk:"ram_unit"`
	DriveSize      types.Int64  `tfsdk:"drive_size"`
	DriveUnit      types.String `tfsdk:"drive_unit"`
	DriveType      types.String `tfsdk:"drive_type"`
	OSName         types.String `tfsdk:"os_name"`
	OSVersion      types.String `tfsdk:"os_version"`
	LocationName   types.String `tfsdk:"location_name"`
	PlanName       types.String `tfsdk:"plan_name"`
	PlanPrice      types.Float64 `tfsdk:"plan_price"`
	CreatedAt      types.String `tfsdk:"created_at"`
}

type AdditionModel struct {
	Code     types.String `tfsdk:"code"`
	Category types.String `tfsdk:"category"`
}

func NewVPSResource() resource.Resource {
	return &VPSResource{}
}

func (r *VPSResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vps"
}

func (r *VPSResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an ISHosting VPS instance.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "VPS instance ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "VPS instance name.",
				Optional:    true,
				Computed:    true,
			},
			"tags": schema.ListAttribute{
				Description: "Tags for the VPS instance.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"plan": schema.StringAttribute{
				Description: "VPS plan code (e.g., 'vps-kvm-lin-1-ber-1m'). Use the ishosting_vps_plans data source to find available plans.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"location": schema.StringAttribute{
				Description: "Location city code (e.g., 'ber' for Berlin). Determined from the plan code.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"auto_renew": schema.BoolAttribute{
				Description: "Whether to auto-renew the VPS.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},

			// Order options
			"os_category": schema.StringAttribute{
				Description: "OS addition category code from plan configs (e.g., 'os_linux_ubuntu').",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"os_code": schema.StringAttribute{
				Description: "OS addition code from plan configs (e.g., 'ubuntu_22_04_64').",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vnc_enabled": schema.BoolAttribute{
				Description: "Enable VNC access.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"ssh_enabled": schema.BoolAttribute{
				Description: "Enable SSH access.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"ssh_keys": schema.ListAttribute{
				Description: "List of SSH key IDs to attach.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"quantity": schema.Int64Attribute{
				Description: "Number of VPS instances to order.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
			},
			"comment": schema.StringAttribute{
				Description: "Comment for the order.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"promos": schema.ListAttribute{
				Description: "Promo codes to apply.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"additions": schema.ListNestedAttribute{
				Description: "Additional configuration options (CPU, RAM, drive, etc.).",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"code": schema.StringAttribute{
							Description: "Addition option code.",
							Required:    true,
						},
						"category": schema.StringAttribute{
							Description: "Addition category code.",
							Required:    true,
						},
					},
				},
			},

			// Internal tracking
			"invoice_id": schema.StringAttribute{
				Description: "Invoice ID from the order. Used to cancel unpaid orders on destroy.",
				Computed:    true,
			},

			// Computed attributes
			"public_ip": schema.StringAttribute{
				Description: "Primary public IP address.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Current VPS status code.",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "Current VPS state code.",
				Computed:    true,
			},
			"platform_name": schema.StringAttribute{
				Description: "Platform name (e.g., Linux, Windows).",
				Computed:    true,
			},
			"cpu_cores": schema.Int64Attribute{
				Description: "Number of CPU cores.",
				Computed:    true,
			},
			"ram_size": schema.Int64Attribute{
				Description: "RAM size.",
				Computed:    true,
			},
			"ram_unit": schema.StringAttribute{
				Description: "RAM unit (e.g., GB).",
				Computed:    true,
			},
			"drive_size": schema.Int64Attribute{
				Description: "Drive size.",
				Computed:    true,
			},
			"drive_unit": schema.StringAttribute{
				Description: "Drive unit (e.g., GB).",
				Computed:    true,
			},
			"drive_type": schema.StringAttribute{
				Description: "Drive type (e.g., SSD, NVMe).",
				Computed:    true,
			},
			"os_name": schema.StringAttribute{
				Description: "Operating system name.",
				Computed:    true,
			},
			"os_version": schema.StringAttribute{
				Description: "Operating system version.",
				Computed:    true,
			},
			"location_name": schema.StringAttribute{
				Description: "Location name.",
				Computed:    true,
			},
			"plan_name": schema.StringAttribute{
				Description: "Plan display name.",
				Computed:    true,
			},
			"plan_price": schema.Float64Attribute{
				Description: "Plan price.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "VPS creation timestamp.",
				Computed:    true,
			},
		},
	}
}

func (r *VPSResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VPSResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan VPSResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build order item
	orderItem := client.OrderItem{
		Action:   "new",
		Type:     "vps",
		Plan:     plan.Plan.ValueString(),
		Quantity: int(plan.Quantity.ValueInt64()),
		Comment:  plan.Comment.ValueString(),
	}
	orderItem.Location.City = plan.Location.ValueString()

	// Options
	options := &client.OrderOptions{}
	options.VNC = &client.OrderVNC{
		IsEnabled: plan.VNCEnabled.ValueBool(),
	}
	options.SSH = &client.OrderSSH{
		IsEnabled: plan.SSHEnabled.ValueBool(),
	}

	// SSH Keys
	if !plan.SSHKeys.IsNull() {
		var sshKeys []string
		resp.Diagnostics.Append(plan.SSHKeys.ElementsAs(ctx, &sshKeys, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		options.SSH.Keys = sshKeys
	}
	orderItem.Options = options

	// Additions
	var additions []client.OrderAddition
	if !plan.Additions.IsNull() {
		var additionModels []AdditionModel
		resp.Diagnostics.Append(plan.Additions.ElementsAs(ctx, &additionModels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, a := range additionModels {
			additions = append(additions, client.OrderAddition{
				Code:     a.Code.ValueString(),
				Category: a.Category.ValueString(),
			})
		}
	}

	// Add OS if specified
	if !plan.OSCode.IsNull() && !plan.OSCategory.IsNull() {
		additions = append(additions, client.OrderAddition{
			Code:     plan.OSCode.ValueString(),
			Category: plan.OSCategory.ValueString(),
		})
	}

	orderItem.Additions = additions

	// Build order request
	orderReq := client.OrderRequest{
		Items: []client.OrderItem{orderItem},
	}

	// Promos
	if !plan.Promos.IsNull() {
		var promos []string
		resp.Diagnostics.Append(plan.Promos.ElementsAs(ctx, &promos, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		orderReq.Promos = promos
	}

	tflog.Debug(ctx, "Creating VPS order")

	// Lock the order mutex to ensure only one order (cart) is processed at a time.
	// Hold the lock until the VPS is active so a concurrent order can't interfere.
	r.client.LockOrder()
	defer r.client.UnlockOrder()

	invoiceResp, err := r.client.CreateOrder(ctx, orderReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating VPS",
			"Could not create VPS order: "+err.Error(),
		)
		return
	}

	// Save invoice ID to state immediately so destroy can cancel it if payment fails
	plan.InvoiceID = types.StringValue(invoiceResp.ID.String())

	// Extract VPS ID from the invoice services
	var vpsID string
	for _, svc := range invoiceResp.Services {
		if svc.Type == "vps" {
			vpsID = svc.Service.ID.String()
			break
		}
	}

	if vpsID == "" {
		// Save state with invoice ID so destroy can cancel the invoice
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError(
			"Error Creating VPS",
			"No VPS service ID returned from order response.",
		)
		return
	}

	// Save state with VPS ID + invoice ID before payment, so destroy can clean up
	plan.ID = types.StringValue(vpsID)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)

	// Pay the invoice
	tflog.Debug(ctx, fmt.Sprintf("Paying invoice %s", invoiceResp.ID.String()))

	payResp, err := r.client.PayInvoice(ctx, invoiceResp.ID.String(), client.PayInvoiceRequest{
		Balance: true,
		Renew:   true,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Paying Invoice",
			fmt.Sprintf("Could not pay invoice %s: %s. Run 'terraform destroy' to cancel the unpaid order.", invoiceResp.ID.String(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Invoice payment response: %s", string(payResp)))
	tflog.Debug(ctx, fmt.Sprintf("VPS ordered and paid, ID: %s.", vpsID))

	// Read VPS so every computed attr is a known value — terraform-plugin-framework
	// refuses the apply otherwise. The VPS is still "installing" at this point;
	// fields like public_ip come back populated, others may be empty strings.
	vps, err := r.client.GetVPS(ctx, vpsID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VPS After Create",
			fmt.Sprintf("VPS %s was paid but could not be read back: %s", vpsID, err.Error()),
		)
		return
	}
	r.mapVPSToModel(vps, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *VPSResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state VPSResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the VPS ID is empty, the order was created but never paid/provisioned.
	// Keep the state as-is so destroy can cancel the invoice.
	if state.ID.IsNull() || state.ID.ValueString() == "" {
		return
	}

	vps, err := r.client.GetVPS(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VPS",
			"Could not read VPS ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	r.mapVPSToModel(vps, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VPSResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan VPSResourceModel
	var state VPSResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	patchReq := client.VPSPatchRequest{}

	// Update name if changed
	if !plan.Name.Equal(state.Name) {
		name := plan.Name.ValueString()
		patchReq.Name = &name
	}

	// Tags are always required by the API in PATCH requests
	var tags []string
	if !plan.Tags.IsNull() {
		resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if tags == nil {
		tags = []string{}
	}
	patchReq.Tags = tags

	// Update auto_renew if changed
	if !plan.AutoRenew.Equal(state.AutoRenew) {
		autoRenew := plan.AutoRenew.ValueBool()
		patchReq.Plan = &struct {
			AutoRenew *bool `json:"auto_renew,omitempty"`
		}{
			AutoRenew: &autoRenew,
		}
	}

	_, err := r.client.UpdateVPS(ctx, state.ID.ValueString(), patchReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating VPS",
			"Could not update VPS "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Read back the full state
	vps, err := r.client.GetVPS(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VPS After Update",
			"Could not read VPS "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	r.mapVPSToModel(vps, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *VPSResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state VPSResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the VPS was never paid for (no state/status), cancel the invoice instead
	if state.State.IsNull() || state.State.ValueString() == "" {
		if !state.InvoiceID.IsNull() && state.InvoiceID.ValueString() != "" {
			tflog.Debug(ctx, fmt.Sprintf("VPS not provisioned, cancelling invoice %s", state.InvoiceID.ValueString()))
			err := r.client.CancelInvoice(ctx, state.InvoiceID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Cancelling Invoice",
					fmt.Sprintf("Could not cancel invoice %s: %s", state.InvoiceID.ValueString(), err.Error()),
				)
				return
			}
			tflog.Debug(ctx, fmt.Sprintf("Invoice %s cancelled", state.InvoiceID.ValueString()))
			return
		}
		// No invoice ID and no VPS state — nothing to clean up
		return
	}

	// ISHosting does not support deleting VPS instances via the API.
	// Instead, disable auto-renew so the instance expires at the end of the billing period.
	autoRenew := false
	patchReq := client.VPSPatchRequest{
		Plan: &struct {
			AutoRenew *bool `json:"auto_renew,omitempty"`
		}{
			AutoRenew: &autoRenew,
		},
	}

	// Tags are required in PATCH requests
	var tags []string
	if !state.Tags.IsNull() {
		state.Tags.ElementsAs(ctx, &tags, false)
	}
	if tags == nil {
		tags = []string{}
	}
	patchReq.Tags = tags

	_, err := r.client.UpdateVPS(ctx, state.ID.ValueString(), patchReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Disabling Auto-Renew",
			"Could not disable auto-renew for VPS "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.AddWarning(
		"VPS Not Deleted",
		fmt.Sprintf("ISHosting does not support deleting VPS instances. Auto-renew has been disabled for VPS %s. "+
			"The instance will be decommissioned at the end of the current billing period.", state.ID.ValueString()),
	)

	tflog.Warn(ctx, fmt.Sprintf("VPS %s: auto-renew disabled, instance will expire at end of billing period", state.ID.ValueString()))
}

func (r *VPSResource) mapVPSToModel(vps *client.VPS, model *VPSResourceModel) {
	model.ID = types.StringValue(vps.ID.String())
	model.Name = types.StringValue(vps.Name)
	model.PublicIP = types.StringValue(vps.Network.PublicIP)
	model.Status = types.StringValue(vps.Status.Code)
	model.State = types.StringValue(vps.Status.State.Code)
	model.PlatformName = types.StringValue(vps.Platform.Name)
	model.OSName = types.StringValue(vps.Platform.Config.OS.Name)
	model.LocationName = types.StringValue(vps.Location.Name)
	model.Location = types.StringValue(strings.ToLower(vps.Location.Code))
	model.PlanName = types.StringValue(vps.Plan.Name)
	model.Plan = types.StringValue(vps.Plan.Code)
	model.AutoRenew = types.BoolValue(vps.Plan.AutoRenew)
	model.CreatedAt = types.StringValue(vps.CreatedAt.String())

	// Config fields arrive as opaque strings; pluck the leading number.
	//   cpu.value  = "2x2900"   → cores=2
	//   ram.value  = "2g"       → size=2
	//   drive.value= "30/nvme"  → size=30
	//   plan.price = "11.99$"   → 11.99
	model.CPUCores = types.Int64Value(leadingInt(vps.Platform.Config.CPU.Value))
	model.RAMSize = types.Int64Value(leadingInt(vps.Platform.Config.RAM.Value))
	model.RAMUnit = types.StringValue(vps.Platform.Config.RAM.Name)
	model.DriveSize = types.Int64Value(leadingInt(vps.Platform.Config.Drive.Value))
	model.DriveUnit = types.StringValue(vps.Platform.Config.Drive.Name)
	model.DriveType = types.StringValue(vps.Platform.Config.Drive.Code)
	model.OSVersion = types.StringValue(vps.Platform.Config.OS.Code)
	model.PlanPrice = types.Float64Value(parsePrice(vps.Plan.Price))

	// Map tags
	if len(vps.Tags) > 0 {
		tags, _ := types.ListValueFrom(context.Background(), types.StringType, vps.Tags)
		model.Tags = tags
	}
}

// leadingInt returns the integer prefix of s, or 0 if s doesn't start with digits.
func leadingInt(s string) int64 {
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0
	}
	n, _ := strconv.ParseInt(s[:end], 10, 64)
	return n
}

// parsePrice strips a trailing "$" and parses the remaining number; returns 0 on failure.
func parsePrice(s string) float64 {
	n, err := strconv.ParseFloat(strings.TrimSuffix(s, "$"), 64)
	if err != nil {
		return 0
	}
	return n
}
