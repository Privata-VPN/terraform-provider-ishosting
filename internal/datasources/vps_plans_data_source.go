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
	_ datasource.DataSource              = &VPSPlansDataSource{}
	_ datasource.DataSourceWithConfigure = &VPSPlansDataSource{}
)

type VPSPlansDataSource struct {
	client *client.Client
}

type VPSPlansDataSourceModel struct {
	Locations types.List  `tfsdk:"locations"`
	Platforms types.List  `tfsdk:"platforms"`
	Plans     []PlanModel `tfsdk:"plans"`
}

type PlanModel struct {
	Name         types.String  `tfsdk:"name"`
	Code         types.String  `tfsdk:"code"`
	Price        types.Float64 `tfsdk:"price"`
	Period       types.String  `tfsdk:"period"`
	LocationName types.String  `tfsdk:"location_name"`
	LocationCode types.String  `tfsdk:"location_code"`
	CityCode     types.String  `tfsdk:"city_code"`
	CityName     types.String  `tfsdk:"city_name"`
	PlatformName types.String  `tfsdk:"platform_name"`
	PlatformCode types.String  `tfsdk:"platform_code"`
	CPUCores     types.Int64   `tfsdk:"cpu_cores"`
	RAMSize      types.Int64   `tfsdk:"ram_size"`
	RAMUnit      types.String  `tfsdk:"ram_unit"`
	DriveSize    types.Int64   `tfsdk:"drive_size"`
	DriveUnit    types.String  `tfsdk:"drive_unit"`
	DriveType    types.String  `tfsdk:"drive_type"`
}

func NewVPSPlansDataSource() datasource.DataSource {
	return &VPSPlansDataSource{}
}

func (d *VPSPlansDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vps_plans"
}

func (d *VPSPlansDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists available ISHosting VPS plans.",
		Attributes: map[string]schema.Attribute{
			"locations": schema.ListAttribute{
				Description: "Filter by location codes.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"platforms": schema.ListAttribute{
				Description: "Filter by platform codes.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"plans": schema.ListNestedAttribute{
				Description: "Available VPS plans.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Plan name.",
							Computed:    true,
						},
						"code": schema.StringAttribute{
							Description: "Plan code (use this in the ishosting_vps resource).",
							Computed:    true,
						},
						"price": schema.Float64Attribute{
							Description: "Plan price.",
							Computed:    true,
						},
						"period": schema.StringAttribute{
							Description: "Billing period.",
							Computed:    true,
						},
						"location_name": schema.StringAttribute{
							Description: "Location country name.",
							Computed:    true,
						},
						"location_code": schema.StringAttribute{
							Description: "Location country code.",
							Computed:    true,
						},
						"city_code": schema.StringAttribute{
							Description: "City code (use this as the location in the ishosting_vps resource).",
							Computed:    true,
						},
						"city_name": schema.StringAttribute{
							Description: "City name.",
							Computed:    true,
						},
						"platform_name": schema.StringAttribute{
							Description: "Platform name (e.g., Linux, Windows).",
							Computed:    true,
						},
						"platform_code": schema.StringAttribute{
							Description: "Platform code.",
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
							Description: "RAM unit.",
							Computed:    true,
						},
						"drive_size": schema.Int64Attribute{
							Description: "Drive size.",
							Computed:    true,
						},
						"drive_unit": schema.StringAttribute{
							Description: "Drive unit.",
							Computed:    true,
						},
						"drive_type": schema.StringAttribute{
							Description: "Drive type.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *VPSPlansDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *VPSPlansDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state VPSPlansDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var locations, platforms []string
	if !state.Locations.IsNull() {
		resp.Diagnostics.Append(state.Locations.ElementsAs(ctx, &locations, false)...)
	}
	if !state.Platforms.IsNull() {
		resp.Diagnostics.Append(state.Platforms.ElementsAs(ctx, &platforms, false)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	plans, err := d.client.ListVPSPlans(ctx, locations, platforms)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VPS Plans",
			"Could not read VPS plans: "+err.Error(),
		)
		return
	}

	state.Plans = make([]PlanModel, len(plans))
	for i, p := range plans {
		state.Plans[i] = PlanModel{
			Name:         types.StringValue(p.Name),
			Code:         types.StringValue(p.Code),
			Price:        types.Float64Value(p.Price),
			Period:       types.StringValue(p.Period),
			LocationName: types.StringValue(p.Location.Name),
			LocationCode: types.StringValue(p.Location.Code),
			CityCode:     types.StringValue(p.Location.Variant.Code),
			CityName:     types.StringValue(p.Location.Variant.Name),
			PlatformName: types.StringValue(p.Platform.Name),
			PlatformCode: types.StringValue(p.Platform.Code),
			CPUCores:     types.Int64Value(int64(p.Platform.Config.CPU.Cores)),
			RAMSize:      types.Int64Value(int64(p.Platform.Config.RAM.Size)),
			RAMUnit:      types.StringValue(p.Platform.Config.RAM.Unit),
			DriveSize:    types.Int64Value(int64(p.Platform.Config.Drive.Size)),
			DriveUnit:    types.StringValue(p.Platform.Config.Drive.Unit),
			DriveType:    types.StringValue(p.Platform.Config.Drive.Type),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
