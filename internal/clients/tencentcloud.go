/*
Copyright 2021 Upbound Inc.
*/

package clients

import (
	"context"
	"encoding/json"
	"os"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/upjet/pkg/terraform"

	"github.com/crossplane-contrib/provider-tencentcloud/apis/v1beta1"
)

const (
	// error messages
	errNoProviderConfig     = "no providerConfigRef provided"
	errGetProviderConfig    = "cannot get referenced ProviderConfig"
	errTrackUsage           = "cannot track ProviderConfig usage"
	errExtractCredentials   = "cannot extract credentials"
	errUnmarshalCredentials = "cannot unmarshal tencentcloud credentials as JSON"
	errMissingWebIdentityEnv = "TKE pod identity requires TKE_ROLE_ARN, TKE_WEB_IDENTITY_TOKEN_FILE, and TKE_PROVIDER_ID env vars"
	keySecretID             = "secret_id"
	keySecretKey            = "secret_key"
	keyRegion               = "region"

	tkeEnvRoleARN              = "TKE_ROLE_ARN"
	tkeEnvWebIdentityTokenFile = "TKE_WEB_IDENTITY_TOKEN_FILE"
	tkeEnvProviderID           = "TKE_PROVIDER_ID"
	tkeEnvRegion               = "TKE_REGION"
	tkeEnvDefaultRegion        = "TKE_DEFAULT_REGION"
)

// TerraformSetupBuilder builds Terraform a terraform.SetupFn function which
// returns Terraform provider setup configuration
func TerraformSetupBuilder(version, providerSource, providerVersion string) terraform.SetupFn {
	return func(ctx context.Context, client client.Client, mg resource.Managed) (terraform.Setup, error) {
		ps := terraform.Setup{
			Version: version,
			Requirement: terraform.ProviderRequirement{
				Source:  providerSource,
				Version: providerVersion,
			},
		}

		configRef := mg.GetProviderConfigReference()
		if configRef == nil {
			return ps, errors.New(errNoProviderConfig)
		}
		pc := &v1beta1.ProviderConfig{}
		if err := client.Get(ctx, types.NamespacedName{Name: configRef.Name}, pc); err != nil {
			return ps, errors.Wrap(err, errGetProviderConfig)
		}

		t := resource.NewProviderConfigUsageTracker(client, &v1beta1.ProviderConfigUsage{})
		if err := t.Track(ctx, mg); err != nil {
			return ps, errors.Wrap(err, errTrackUsage)
		}

		ps.Configuration = map[string]interface{}{}

		if pc.Spec.Credentials.Source == xpv1.CredentialsSourceInjectedIdentity {
			roleARN := os.Getenv(tkeEnvRoleARN)
			tokenFile := os.Getenv(tkeEnvWebIdentityTokenFile)
			providerID := os.Getenv(tkeEnvProviderID)
			if roleARN == "" || tokenFile == "" || providerID == "" {
				return ps, errors.New(errMissingWebIdentityEnv)
			}
			tokenBytes, err := os.ReadFile(tokenFile)
			if err != nil {
				return ps, errors.Wrap(err, "cannot read web identity token file")
			}
			ps.Configuration["assume_role_with_web_identity"] = []interface{}{
				map[string]interface{}{
					"provider_id":        providerID,
					"role_arn":           roleARN,
					"session_name":       "crossplane",
					"session_duration":   3600,
					"web_identity_token": string(tokenBytes),
				},
			}
			region := os.Getenv(tkeEnvRegion)
			if region == "" {
				region = os.Getenv(tkeEnvDefaultRegion)
			}
			if region != "" {
				ps.Configuration[keyRegion] = region
			}
			return ps, nil
		}

		data, err := resource.CommonCredentialExtractor(ctx, pc.Spec.Credentials.Source, client, pc.Spec.Credentials.CommonCredentialSelectors)
		if err != nil {
			return ps, errors.Wrap(err, errExtractCredentials)
		}
		creds := map[string]string{}
		if err := json.Unmarshal(data, &creds); err != nil {
			return ps, errors.Wrap(err, errUnmarshalCredentials)
		}

		if v, ok := creds[keySecretID]; ok {
			ps.Configuration[keySecretID] = v
		}
		if v, ok := creds[keySecretKey]; ok {
			ps.Configuration[keySecretKey] = v
		}
		if v, ok := creds[keyRegion]; ok {
			ps.Configuration[keyRegion] = v
		}
		return ps, nil
	}
}
