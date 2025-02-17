package sshkey

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var frameworkDatasourceSchema = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"label": schema.StringAttribute{
			Description: "The label of the Linode SSH Key.",
			Required:    true,
		},
		"ssh_key": schema.StringAttribute{
			Description: "The public SSH Key, which is used to authenticate to the root user of the Linodes you deploy.",
			Computed:    true,
		},
		"created": schema.StringAttribute{
			Description: "The date this key was added.",
			Computed:    true,
		},
		"id": schema.StringAttribute{
			Description: "A unique identifier for this datasource.",
			Optional:    true,
		},
	},
}
