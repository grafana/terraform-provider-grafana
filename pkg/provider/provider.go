package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"
	"github.com/hashicorp/terraform-plugin-mux/tf6to5server"
)

func MakeProviderServer(ctx context.Context, version string) (tfprotov5.ProviderServer, error) {
	// While we still have the SDK2 provider, we have to use the provider v5 protocol
	// See https://developer.hashicorp.com/terraform/plugin/mux/translating-protocol-version-6-to-5
	downgradedFrameworkProvider, err := tf6to5server.DowngradeServer(
		context.Background(),
		providerserver.NewProtocol6(FrameworkProvider(version)),
	)
	if err != nil {
		return nil, err
	}

	providers := []func() tfprotov5.ProviderServer{
		func() tfprotov5.ProviderServer {
			return downgradedFrameworkProvider
		},
		Provider(version).GRPCProvider,
	}
	muxServer, err := tf5muxserver.NewMuxServer(ctx, providers...)
	if err != nil {
		return nil, err
	}
	return muxServer, nil
}
