package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
)

func MakeProviderServer(ctx context.Context, version string) (tfprotov6.ProviderServer, error) {
	// While we still have the SDK2 provider, we upgrade it to protocol v6 so it can be muxed with the framework provider
	// See https://developer.hashicorp.com/terraform/plugin/mux/translating-protocol-version-5-to-6
	upgradedSDKProvider, err := tf5to6server.UpgradeServer(
		ctx,
		Provider(version).GRPCProvider,
	)
	if err != nil {
		return nil, err
	}

	providers := []func() tfprotov6.ProviderServer{
		providerserver.NewProtocol6(FrameworkProvider(version)),
		func() tfprotov6.ProviderServer {
			return upgradedSDKProvider
		},
	}
	muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)
	if err != nil {
		return nil, err
	}
	return muxServer, nil
}
