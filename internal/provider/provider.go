package provider

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rayval1/terraform-provider-nodegrid/internal/client"
)

// ProviderConfig carries the credentials shared by every device resource;
// the device IP itself lives on each resource so one provider block covers
// a fleet of devices.
type ProviderConfig struct {
	Username string
	Password string
	Port     int
	Timeout  time.Duration
}

func (p ProviderConfig) ClientFor(host string) *client.Client {
	return client.New(client.Config{
		Host:     host,
		Port:     p.Port,
		Username: p.Username,
		Password: p.Password,
		Timeout:  p.Timeout,
	})
}

type nodegridProvider struct {
	version string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &nodegridProvider{version: version}
	}
}

type providerModel struct {
	Username       types.String `tfsdk:"username"`
	Password       types.String `tfsdk:"password"`
	Port           types.Int64  `tfsdk:"port"`
	TimeoutSeconds types.Int64  `tfsdk:"timeout_seconds"`
}

func (p *nodegridProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "nodegrid"
	resp.Version = p.version
}

func (p *nodegridProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages ZPE Nodegrid devices over SSH using the Nodegrid CLI settings tree.",
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Optional:    true,
				Description: "SSH/CLI user. Falls back to NODEGRID_USERNAME, then \"admin\".",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "SSH/CLI password. Falls back to NODEGRID_PASSWORD.",
			},
			"port": schema.Int64Attribute{
				Optional:    true,
				Description: "SSH port, default 22.",
			},
			"timeout_seconds": schema.Int64Attribute{
				Optional:    true,
				Description: "Per-session SSH timeout, default 30.",
			},
		},
	}
}

func (p *nodegridProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var model providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg := ProviderConfig{
		Username: model.Username.ValueString(),
		Password: model.Password.ValueString(),
		Port:     int(model.Port.ValueInt64()),
		Timeout:  time.Duration(model.TimeoutSeconds.ValueInt64()) * time.Second,
	}
	if cfg.Username == "" {
		cfg.Username = os.Getenv("NODEGRID_USERNAME")
	}
	if cfg.Username == "" {
		cfg.Username = "admin"
	}
	if cfg.Password == "" {
		cfg.Password = os.Getenv("NODEGRID_PASSWORD")
	}
	if cfg.Port == 0 {
		if v, err := strconv.Atoi(os.Getenv("NODEGRID_SSH_PORT")); err == nil {
			cfg.Port = v
		}
	}

	if cfg.Password == "" {
		resp.Diagnostics.AddError(
			"Missing Nodegrid password",
			"Set the provider password attribute or the NODEGRID_PASSWORD environment variable.",
		)
		return
	}

	resp.ResourceData = cfg
	resp.DataSourceData = cfg
}

func (p *nodegridProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSettingsResource,
		NewExecResource,
	}
}

func (p *nodegridProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSettingsDataSource,
	}
}
