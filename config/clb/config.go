/*
Copyright 2021 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clb

import (
	"context"
	"fmt"

	"github.com/crossplane/upjet/pkg/config"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const shortGroupClb = "clb"

// Configure configures the clb group
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("tencentcloud_clb_instance", func(r *config.Resource) {
		r.ShortGroup = shortGroupClb
		r.Kind = "Instance"
		r.References["vpc_id"] = config.Reference{
			Type: "github.com/crossplane-contrib/provider-tencentcloud/apis/vpc/v1alpha1.VPC",
		}
		r.References["subnet_id"] = config.Reference{
			Type: "github.com/crossplane-contrib/provider-tencentcloud/apis/vpc/v1alpha1.Subnet",
		}
	})

	p.AddResourceConfigurator("tencentcloud_clb_listener", func(r *config.Resource) {
		r.ShortGroup = shortGroupClb
		r.Kind = "Listener"
		r.References["clb_id"] = config.Reference{
			Type: "Instance",
		}
	})

	p.AddResourceConfigurator("tencentcloud_clb_listener_rule", func(r *config.Resource) {
		r.ShortGroup = shortGroupClb
		r.Kind = "ListenerRule"
		r.References["clb_id"] = config.Reference{
			Type: "Instance",
		}
		r.References["listener_id"] = config.Reference{
			Type:      "Listener",
			Extractor: "github.com/crossplane/upjet/pkg/resource.ExtractParamPath(\"listener_id\",true)",
		}
		// domain and domains are mutually exclusive: the API accepts only one.
		// The upstream schema does not declare ConflictsWith, so both fields can
		// be populated simultaneously (e.g. domain late-initialized from the API
		// response while domains is set in spec), causing refresh to fail.
		if s, ok := r.TerraformResource.Schema["domain"]; ok {
			s.ConflictsWith = []string{"domains"}
		}
		if s, ok := r.TerraformResource.Schema["domains"]; ok {
			s.ConflictsWith = []string{"domain"}
		}
		// Prevent late-init from copying whichever field the user did NOT set.
		// The API echoes both domain and domains in its response; without this
		// guard, the unused field gets injected into spec and triggers the
		// ConflictsWith error on the next reconcile.
		r.LateInitializer = config.LateInitializer{
			IgnoredFields: []string{"domain", "domains"},
		}
	})

	p.AddResourceConfigurator("tencentcloud_clb_attachment", func(r *config.Resource) {
		r.ShortGroup = shortGroupClb
		r.Kind = "Attachment"
		r.References["clb_id"] = config.Reference{
			Type: "Instance",
		}
		r.References["listener_id"] = config.Reference{
			Type:      "Listener",
			Extractor: "github.com/crossplane/upjet/pkg/resource.ExtractParamPath(\"listener_id\",true)",
		}
		r.References["rule_id"] = config.Reference{
			Type:      "ListenerRule",
			Extractor: "github.com/crossplane/upjet/pkg/resource.ExtractParamPath(\"rule_id\",true)",
		}
	})

	p.AddResourceConfigurator("tencentcloud_clb_customized_config", func(r *config.Resource) {
		r.ShortGroup = shortGroupClb
		r.Kind = "CustomizedConfig"
	})

	p.AddResourceConfigurator("tencentcloud_clb_log_set", func(r *config.Resource) {
		r.ShortGroup = shortGroupClb
		r.Kind = "LogSet"
	})

	p.AddResourceConfigurator("tencentcloud_clb_log_topic", func(r *config.Resource) {
		r.ShortGroup = shortGroupClb
		r.Kind = "LogTopic"
		r.References["log_set_id"] = config.Reference{
			Type: "LogSet",
		}
	})

	p.AddResourceConfigurator("tencentcloud_clb_redirection", func(r *config.Resource) {
		r.ShortGroup = shortGroupClb
		r.Kind = "Redirection"
		r.References["clb_id"] = config.Reference{
			Type: "Instance",
		}
		r.References["source_listener_id"] = config.Reference{
			Type:      "Listener",
			Extractor: "github.com/crossplane/upjet/pkg/resource.ExtractParamPath(\"listener_id\",true)",
		}
		r.References["target_listener_id"] = config.Reference{
			Type:      "Listener",
			Extractor: "github.com/crossplane/upjet/pkg/resource.ExtractParamPath(\"listener_id\",true)",
		}
		r.References["source_rule_id"] = config.Reference{
			Type: "ListenerRule",
		}
		r.References["target_rule_id"] = config.Reference{
			Type: "ListenerRule",
		}
	})

	p.AddResourceConfigurator("tencentcloud_clb_snat_ip", func(r *config.Resource) {
		r.ShortGroup = shortGroupClb
		r.Kind = "SnatIp"
		r.References["clb_id"] = config.Reference{
			Type: "Instance",
		}
	})

	p.AddResourceConfigurator("tencentcloud_clb_target_group", func(r *config.Resource) {
		r.ShortGroup = shortGroupClb
		r.Kind = "TargetGroup"
		r.References["vpc_id"] = config.Reference{
			Type: "github.com/crossplane-contrib/provider-tencentcloud/apis/vpc/v1alpha1.VPC",
		}
		// Protocol is required by TencentCloud API when type=v2 but the upstream
		// Terraform schema marks it optional. Enforce it here to surface the error
		// at plan time rather than getting a cryptic API error during apply.
		//
		// type is ForceNew: the TencentCloud API does not support in-place type
		// changes (v1→v2). Without ForceNew the controller would send a no-op
		// update, leaving a v1 TG on the API side and causing
		// "not targetgroup v2 mode" errors on TargetGroupAttachment apply.
		r.TerraformResource.CustomizeDiff = customdiff.All(
			r.TerraformResource.CustomizeDiff,
			func(_ context.Context, d *schema.ResourceDiff, _ any) error {
				t, _ := d.GetOk("type")
				p, _ := d.GetOk("protocol")
				if t == "v2" && (p == nil || p == "") {
					return fmt.Errorf("protocol is required when type is v2")
				}
				return nil
			},
			customdiff.ForceNewIfChange("type", func(_ context.Context, old, new, _ interface{}) bool {
				return old != new
			}),
		)
	})

	p.AddResourceConfigurator("tencentcloud_clb_target_group_attachment", func(r *config.Resource) {
		r.ShortGroup = shortGroupClb
		r.Kind = "TargetGroupAttachment"
		r.References["clb_id"] = config.Reference{
			Type: "Instance",
		}
		r.References["listener_id"] = config.Reference{
			Type:      "Listener",
			Extractor: "github.com/crossplane/upjet/pkg/resource.ExtractParamPath(\"listener_id\",true)",
		}
		r.References["rule_id"] = config.Reference{
			Type:      "ListenerRule",
			Extractor: "github.com/crossplane/upjet/pkg/resource.ExtractParamPath(\"rule_id\",true)",
		}
		r.References["target_group_id"] = config.Reference{
			Type: "TargetGroup",
		}
	})

	p.AddResourceConfigurator("tencentcloud_clb_target_group_instance_attachment", func(r *config.Resource) {
		r.ShortGroup = shortGroupClb
		r.Kind = "TargetGroupInstanceAttachment"
		r.References["target_group_id"] = config.Reference{
			Type: "TargetGroup",
		}
	})

	p.AddResourceConfigurator("tencentcloud_lb", func(r *config.Resource) {
		r.ShortGroup = shortGroupClb
		r.Kind = "LB"
	})

	p.AddResourceConfigurator("tencentcloud_alb_server_attachment", func(r *config.Resource) {
		r.ShortGroup = shortGroupClb
		r.Kind = "AlbServerAttachment"
	})
}
