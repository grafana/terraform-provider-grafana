package appplatform

import (
	"testing"

	provisioningv0alpha1 "github.com/grafana/grafana/apps/provisioning/pkg/apis/provisioning/v0alpha1"
	apicommon "github.com/grafana/grafana/pkg/apimachinery/apis/common/v0alpha1"
	"github.com/stretchr/testify/require"
)

func TestProvisioningConnectionSecureSubresourceAccessors(t *testing.T) {
	obj := &ProvisioningConnection{}
	secure := provisioningv0alpha1.ConnectionSecure{
		PrivateKey:   apicommon.InlineSecureValue{Create: apicommon.NewSecretValue("raw-private-key")},
		ClientSecret: apicommon.InlineSecureValue{Remove: true},
		Token:        apicommon.InlineSecureValue{Name: "token-ref"},
	}

	require.NoError(t, obj.SetSubresource("secure", secure))
	require.Equal(t, secure, obj.Secure)

	got, ok := obj.GetSubresource("secure")
	require.True(t, ok)
	require.Equal(t, secure, got)

	subresources := obj.GetSubresources()
	require.Equal(t, map[string]any{
		"secure": map[string]any{
			"privateKey": map[string]any{
				"create": "raw-private-key",
			},
			"clientSecret": map[string]any{
				"remove": true,
			},
			"token": map[string]any{
				"name": "token-ref",
			},
		},
	}, subresources)

	_, ok = obj.GetSubresource("unknown")
	require.False(t, ok)
	require.ErrorContains(t, obj.SetSubresource("unknown", secure), "does not exist")
	require.ErrorContains(t, obj.SetSubresource("secure", "invalid"), "not of type ConnectionSecure")
}

func TestProvisioningRepositorySecureSubresourceAccessors(t *testing.T) {
	obj := &ProvisioningRepository{}
	secure := provisioningv0alpha1.SecureValues{
		Token:         apicommon.InlineSecureValue{Create: apicommon.NewSecretValue("raw-token")},
		WebhookSecret: apicommon.InlineSecureValue{Name: "hook-ref"},
	}

	require.NoError(t, obj.SetSubresource("secure", secure))
	require.Equal(t, secure, obj.Secure)

	got, ok := obj.GetSubresource("secure")
	require.True(t, ok)
	require.Equal(t, secure, got)

	subresources := obj.GetSubresources()
	require.Equal(t, map[string]any{
		"secure": map[string]any{
			"token": map[string]any{
				"create": "raw-token",
			},
			"webhookSecret": map[string]any{
				"name": "hook-ref",
			},
		},
	}, subresources)

	_, ok = obj.GetSubresource("unknown")
	require.False(t, ok)
	require.ErrorContains(t, obj.SetSubresource("unknown", secure), "does not exist")
	require.ErrorContains(t, obj.SetSubresource("secure", "invalid"), "not of type SecureValues")
}

func TestSecureSubresourceSupportAccessors(t *testing.T) {
	obj := &secureSubresourceSupport[provisioningv0alpha1.ConnectionSecure]{}
	secure := provisioningv0alpha1.ConnectionSecure{
		PrivateKey:   apicommon.InlineSecureValue{Create: apicommon.NewSecretValue("raw-private-key")},
		ClientSecret: apicommon.InlineSecureValue{Remove: true},
		Token:        apicommon.InlineSecureValue{Name: "token-ref"},
	}

	require.NoError(t, obj.SetSubresource("secure", secure))
	require.Equal(t, secure, obj.Secure)

	got, ok := obj.GetSubresource("secure")
	require.True(t, ok)
	require.Equal(t, secure, got)

	require.Equal(t, map[string]any{
		"secure": map[string]any{
			"privateKey": map[string]any{
				"create": "raw-private-key",
			},
			"clientSecret": map[string]any{
				"remove": true,
			},
			"token": map[string]any{
				"name": "token-ref",
			},
		},
	}, obj.GetSubresources())

	_, ok = obj.GetSubresource("unknown")
	require.False(t, ok)
	require.ErrorContains(t, obj.SetSubresource("unknown", secure), "does not exist")
	require.ErrorContains(t, obj.SetSubresource("secure", "invalid"), "not of type ConnectionSecure")
}

func TestAddSecureSubresourceMergesWithExistingSubresources(t *testing.T) {
	secure := provisioningv0alpha1.SecureValues{
		Token: apicommon.InlineSecureValue{Create: apicommon.NewSecretValue("raw-token")},
	}

	subresources := addSecureSubresource(map[string]any{
		"status": "ok",
	}, secure)

	require.Equal(t, map[string]any{
		"status": "ok",
		"secure": map[string]any{
			"token": map[string]any{
				"create": "raw-token",
			},
		},
	}, subresources)
}

func TestSetSecureSubresourceHelper(t *testing.T) {
	var secure provisioningv0alpha1.SecureValues

	handled, err := setSecureSubresource("status", "ignored", &secure)
	require.False(t, handled)
	require.NoError(t, err)

	handled, err = setSecureSubresource("secure", provisioningv0alpha1.SecureValues{
		Token: apicommon.InlineSecureValue{Name: "token-ref"},
	}, &secure)
	require.True(t, handled)
	require.NoError(t, err)
	require.Equal(t, apicommon.InlineSecureValue{Name: "token-ref"}, secure.Token)

	handled, err = setSecureSubresource("secure", "invalid", &secure)
	require.True(t, handled)
	require.ErrorContains(t, err, "not of type SecureValues")
}

type secureSubresourcePayloadTestModel struct {
	Token  apicommon.InlineSecureValue `json:"token,omitempty"`
	Hidden apicommon.InlineSecureValue `json:"-"`
	Other  string                      `json:"other,omitempty"`
}

func TestSecureSubresourcePayloadStructFiltersUnsupportedAndEmptyFields(t *testing.T) {
	payload := secureSubresourcePayload(secureSubresourcePayloadTestModel{
		Token:  apicommon.InlineSecureValue{Name: "token-ref"},
		Hidden: apicommon.InlineSecureValue{Name: "hidden-ref"},
		Other:  "ignored",
	})

	require.Equal(t, map[string]any{
		"token": map[string]any{
			"name": "token-ref",
		},
	}, payload)
}

func TestSecureSubresourcePayloadMapFiltersEmptyFields(t *testing.T) {
	payload := secureSubresourcePayload(map[string]apicommon.InlineSecureValue{
		"token":  {Remove: true},
		"unused": {},
	})

	require.Equal(t, map[string]any{
		"token": map[string]any{
			"remove": true,
		},
	}, payload)
}

func TestAddSecureSubresourceReturnsEmptyForNilInput(t *testing.T) {
	var secure *provisioningv0alpha1.SecureValues
	require.Equal(t, map[string]any{}, addSecureSubresource(nil, secure))
}
