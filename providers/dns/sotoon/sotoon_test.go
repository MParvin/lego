package sotoon

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-acme/lego/v5/internal/tester"
	"github.com/go-acme/lego/v5/internal/tester/servermock"
	"github.com/stretchr/testify/require"
)

const envDomain = envNamespace + "DOMAIN"

var envTest = tester.NewEnvTest(
	EnvToken,
	EnvWorkspaceUUID,
).WithDomain(envDomain)

func TestNewDNSProvider(t *testing.T) {
	testCases := []struct {
		desc     string
		envVars  map[string]string
		expected string
	}{
		{
			desc: "success",
			envVars: map[string]string{
				EnvToken:         "secret",
				EnvWorkspaceUUID: "workspace-uuid",
			},
		},
		{
			desc: "missing token",
			envVars: map[string]string{
				EnvToken:         "",
				EnvWorkspaceUUID: "workspace-uuid",
			},
			expected: "sotoon: some credentials information are missing: SOTOON_TOKEN",
		},
		{
			desc: "missing workspace UUID",
			envVars: map[string]string{
				EnvToken:         "secret",
				EnvWorkspaceUUID: "",
			},
			expected: "sotoon: some credentials information are missing: SOTOON_WORKSPACE_UUID",
		},
		{
			desc:     "missing credentials",
			envVars:  map[string]string{},
			expected: "sotoon: some credentials information are missing: SOTOON_TOKEN,SOTOON_WORKSPACE_UUID",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			defer envTest.RestoreEnv()

			envTest.ClearEnv()

			envTest.Apply(test.envVars)

			p, err := NewDNSProvider()

			if test.expected == "" {
				require.NoError(t, err)
				require.NotNil(t, p)
				require.NotNil(t, p.config)
				require.NotNil(t, p.client)
			} else {
				require.EqualError(t, err, test.expected)
			}
		})
	}
}

func TestNewDNSProviderConfig(t *testing.T) {
	testCases := []struct {
		desc          string
		token         string
		workspaceUUID string
		expected      string
	}{
		{
			desc:          "success",
			token:         "secret",
			workspaceUUID: "workspace-uuid",
		},
		{
			desc:          "missing token",
			workspaceUUID: "workspace-uuid",
			expected:      "sotoon: credentials missing",
		},
		{
			desc:     "missing workspace UUID",
			token:    "secret",
			expected: "sotoon: credentials missing",
		},
		{
			desc:     "missing credentials",
			expected: "sotoon: credentials missing",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			config := NewDefaultConfig()
			config.Token = test.token
			config.WorkspaceUUID = test.workspaceUUID

			p, err := NewDNSProviderConfig(config)

			if test.expected == "" {
				require.NoError(t, err)
				require.NotNil(t, p)
				require.NotNil(t, p.config)
				require.NotNil(t, p.client)
			} else {
				require.EqualError(t, err, test.expected)
			}
		})
	}
}

func TestLivePresent(t *testing.T) {
	if !envTest.IsLiveTest() {
		t.Skip("skipping live test")
	}

	envTest.RestoreEnv()

	provider, err := NewDNSProvider()
	require.NoError(t, err)

	err = provider.Present(t.Context(), envTest.GetDomain(), "", "123d==")
	require.NoError(t, err)
}

func TestLiveCleanUp(t *testing.T) {
	if !envTest.IsLiveTest() {
		t.Skip("skipping live test")
	}

	envTest.RestoreEnv()

	provider, err := NewDNSProvider()
	require.NoError(t, err)

	err = provider.CleanUp(t.Context(), envTest.GetDomain(), "", "123d==")
	require.NoError(t, err)
}

func mockBuilder() *servermock.Builder[*DNSProvider] {
	return servermock.NewBuilder(
		func(server *httptest.Server) (*DNSProvider, error) {
			config := NewDefaultConfig()
			config.Token = "secret"
			config.WorkspaceUUID = "workspace-uuid"
			config.HTTPClient = server.Client()

			p, err := NewDNSProviderConfig(config)
			if err != nil {
				return nil, err
			}

			p.client.BaseURL, _ = url.Parse(server.URL)

			return p, nil
		},
		servermock.CheckHeader().
			WithAccept("application/json").
			WithAuthorization("Bearer secret"),
	)
}

func TestDNSProvider_Present(t *testing.T) {
	provider := mockBuilder().
		Route("GET /workspaces/workspace-uuid/domainzones",
			servermock.ResponseFromInternal("list_domainzones.json"),
		).
		Route("GET /workspaces/workspace-uuid/domainzones/example-com",
			servermock.ResponseFromInternal("get_domainzone.json"),
		).
		Route("PATCH /workspaces/workspace-uuid/domainzones/example-com",
			servermock.ResponseFromInternal("get_domainzone_with_txt.json"),
			servermock.CheckHeader().
				WithContentType("application/merge-patch+json").
				WithAccept("application/json").
				WithAuthorization("Bearer secret"),
			servermock.CheckRequestJSONBodyFromInternal("patch_add_txt-request.json"),
		).
		Build(t)

	err := provider.Present(t.Context(), "example.com", "abc", "123d==")
	require.NoError(t, err)
}

func TestDNSProvider_Present_alreadyExists(t *testing.T) {
	provider := mockBuilder().
		Route("GET /workspaces/workspace-uuid/domainzones",
			servermock.ResponseFromInternal("list_domainzones.json"),
		).
		Route("GET /workspaces/workspace-uuid/domainzones/example-com",
			servermock.ResponseFromInternal("get_domainzone_with_txt.json"),
		).
		Build(t)

	err := provider.Present(t.Context(), "example.com", "abc", "123d==")
	require.NoError(t, err)
}

func TestDNSProvider_CleanUp(t *testing.T) {
	provider := mockBuilder().
		Route("GET /workspaces/workspace-uuid/domainzones",
			servermock.ResponseFromInternal("list_domainzones.json"),
		).
		Route("GET /workspaces/workspace-uuid/domainzones/example-com",
			servermock.ResponseFromInternal("get_domainzone_with_txt.json"),
		).
		Route("PATCH /workspaces/workspace-uuid/domainzones/example-com",
			servermock.ResponseFromInternal("get_domainzone.json"),
			servermock.CheckHeader().
				WithContentType("application/merge-patch+json").
				WithAccept("application/json").
				WithAuthorization("Bearer secret"),
			servermock.CheckRequestJSONBodyFromInternal("patch_remove_txt-request.json"),
		).
		Build(t)

	err := provider.CleanUp(t.Context(), "example.com", "abc", "123d==")
	require.NoError(t, err)
}

func TestDNSProvider_CleanUp_missingRecord(t *testing.T) {
	provider := mockBuilder().
		Route("GET /workspaces/workspace-uuid/domainzones",
			servermock.ResponseFromInternal("list_domainzones.json"),
		).
		Route("GET /workspaces/workspace-uuid/domainzones/example-com",
			servermock.ResponseFromInternal("get_domainzone.json"),
		).
		Build(t)

	err := provider.CleanUp(t.Context(), "example.com", "abc", "123d==")
	require.NoError(t, err)
}
