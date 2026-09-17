package internal

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-acme/lego/v5/internal/tester/servermock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testToken         = "secret"
	testWorkspaceUUID = "workspace-uuid"
)

func mockBuilder() *servermock.Builder[*Client] {
	return servermock.NewBuilder[*Client](
		func(server *httptest.Server) (*Client, error) {
			client, err := NewClient(testToken, testWorkspaceUUID)
			if err != nil {
				return nil, err
			}

			client.BaseURL, _ = url.Parse(server.URL)
			client.HTTPClient = server.Client()

			return client, nil
		},
		servermock.CheckHeader().
			WithAccept("application/json").
			WithAuthorization("Bearer "+testToken),
	)
}

func TestNewClient(t *testing.T) {
	testCases := []struct {
		desc          string
		token         string
		workspaceUUID string
		expected      string
	}{
		{
			desc:          "success",
			token:         "token",
			workspaceUUID: "uuid",
		},
		{
			desc:          "missing token",
			workspaceUUID: "uuid",
			expected:      "credentials missing",
		},
		{
			desc:     "missing workspace UUID",
			token:    "token",
			expected: "credentials missing",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			client, err := NewClient(test.token, test.workspaceUUID)

			if test.expected == "" {
				require.NoError(t, err)
				require.NotNil(t, client)
			} else {
				require.EqualError(t, err, test.expected)
			}
		})
	}
}

func TestClient_ListDomainZones(t *testing.T) {
	client := mockBuilder().
		Route("GET /workspaces/"+testWorkspaceUUID+"/domainzones",
			servermock.ResponseFromFixture("list_domainzones.json"),
		).
		Build(t)

	zones, err := client.ListDomainZones(t.Context())
	require.NoError(t, err)

	require.Len(t, zones, 2)
	assert.Equal(t, "example-com", zones[0].Metadata.Name)
	assert.Equal(t, "example.com", zones[0].Spec.Origin)
	assert.Equal(t, "other-example-com", zones[1].Metadata.Name)
	assert.Equal(t, "other.example.com", zones[1].Spec.Origin)
}

func TestClient_GetDomainZone(t *testing.T) {
	client := mockBuilder().
		Route("GET /workspaces/"+testWorkspaceUUID+"/domainzones/example-com",
			servermock.ResponseFromFixture("get_domainzone.json"),
		).
		Build(t)

	zone, err := client.GetDomainZone(t.Context(), "example-com")
	require.NoError(t, err)

	assert.Equal(t, "example-com", zone.Metadata.Name)
	assert.Equal(t, "example.com", zone.Spec.Origin)
}

func TestClient_PatchDomainZone(t *testing.T) {
	client := mockBuilder().
		Route("PATCH /workspaces/"+testWorkspaceUUID+"/domainzones/example-com",
			servermock.ResponseFromFixture("get_domainzone_with_txt.json"),
			servermock.CheckHeader().
				WithContentType("application/merge-patch+json").
				WithAccept("application/json").
				WithAuthorization("Bearer "+testToken),
			servermock.CheckRequestJSONBodyFromFixture("patch_add_txt-request.json"),
		).
		Build(t)

	patch := DomainZonePatch{
		Spec: SpecPatch{
			Records: map[string]any{
				"_acme-challenge": []Record{
					{
						TXT:  "ADw2sEd82DUgXcQ9hNBZThJs7zVJkR5v9JeSbAb9mZY",
						Type: "TXT",
						TTL:  120,
					},
				},
			},
		},
	}

	zone, err := client.PatchDomainZone(t.Context(), "example-com", patch)
	require.NoError(t, err)

	require.NotNil(t, zone)
	assert.Equal(t, "example.com", zone.Spec.Origin)
	require.Contains(t, zone.Spec.Records, "_acme-challenge")
	assert.Equal(t, "ADw2sEd82DUgXcQ9hNBZThJs7zVJkR5v9JeSbAb9mZY", zone.Spec.Records["_acme-challenge"][0].TXT)
}

func TestClient_PatchDomainZone_error(t *testing.T) {
	client := mockBuilder().
		Route("PATCH /workspaces/"+testWorkspaceUUID+"/domainzones/example-com",
			servermock.ResponseFromFixture("error.json").
				WithStatusCode(http.StatusUnauthorized),
			servermock.CheckHeader().
				WithContentType("application/merge-patch+json").
				WithAccept("application/json").
				WithAuthorization("Bearer "+testToken),
		).
		Build(t)

	patch := DomainZonePatch{
		Spec: SpecPatch{
			Records: map[string]any{
				"_acme-challenge": nil,
			},
		},
	}

	_, err := client.PatchDomainZone(t.Context(), "example-com", patch)
	require.EqualError(t, err, "Unauthorized: Unauthorized")
}
