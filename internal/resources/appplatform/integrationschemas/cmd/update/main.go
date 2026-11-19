// Command update regenerates integration-schemas.json from the integration type schemas in a given
// commit of grafana/grafana, and reduces them to what the provider uses. Run it from the repository
// root. See ../../README.md.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"time"

	"github.com/grafana/grafana/apps/alerting/notifications/pkg/apis/alertingnotifications/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform/integrationschemas"
)

const (
	repository   = "grafana/grafana"
	upstreamPath = "pkg/tests/apis/alerting/notifications/integrationtypeschema/test-data/list.json"
	snapshotPath = "internal/resources/appplatform/integrationschemas/integration-schemas.json"
	maxSize      = 32 << 20
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	sha := flag.String("sha", "", "commit of "+repository+" to take the integration type schemas from (40 hex digits, required)")
	file := flag.String("file", "", "read the schemas from this local copy of the file at that commit, instead of downloading it")
	flag.Parse()

	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(*sha) {
		return fmt.Errorf("-sha must be a full commit SHA of %s", repository)
	}

	var upstream []byte
	var err error
	if *file != "" {
		upstream, err = os.ReadFile(*file)
	} else {
		upstream, err = download(fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", repository, *sha, upstreamPath))
	}
	if err != nil {
		return err
	}

	reduced, err := reduce(upstream, integrationschemas.Source{Repository: repository, SHA: *sha, Path: upstreamPath})
	if err != nil {
		return err
	}
	if err := os.WriteFile(snapshotPath, reduced, 0o600); err != nil { // #nosec G703 -- the path is a constant
		return err
	}
	fmt.Printf("wrote %s from %s@%s\n", snapshotPath, repository, *sha)
	return nil
}

func download(url string) ([]byte, error) {
	client := &http.Client{Timeout: time.Minute}
	resp, err := client.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: %s", url, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize+1))
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", url, err)
	}
	if len(data) > maxSize {
		return nil, fmt.Errorf("downloading %s: larger than %d bytes", url, maxSize)
	}
	return data, nil
}

const (
	supportedAPIVersion = "notifications.alerting.grafana.app/v1beta1"
	listKind            = "IntegrationTypeSchemaList"
	itemKind            = "IntegrationTypeSchema"
)

// The API's types, under shorter names.
type (
	upstreamType   = v1beta1.GetIntegrationtypeschemasIntegrationTypeSchema
	upstreamField  = v1beta1.GetIntegrationtypeschemasField
	upstreamOption = v1beta1.GetIntegrationtypeschemasSelectOption
)

// upstreamItem is a list item: the API client's type, plus the type meta it does not declare.
type upstreamItem struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	v1beta1.GetIntegrationtypeschemasIntegrationTypeSchemaResource
}

type upstreamList struct {
	APIVersion string         `json:"apiVersion"`
	Kind       string         `json:"kind"`
	Metadata   map[string]any `json:"metadata"`
	Items      []upstreamItem `json:"items"`
}

// reduce converts the list response of Grafana's IntegrationTypeSchema API into the snapshot that is
// committed: only the integration types and versions that can be created, and only what this package
// uses of each field. source says where the response comes from.
//
// It fails on an unexpected API version or kind, on keys the API client's types do not know (which
// means the client needs updating), and on field kinds this package cannot represent, rather than
// dropping them.
func reduce(upstream []byte, source integrationschemas.Source) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(upstream))
	dec.DisallowUnknownFields()

	var l upstreamList
	if err := dec.Decode(&l); err != nil {
		return nil, fmt.Errorf("decoding integration type schemas: %w", err)
	}
	if l.APIVersion != supportedAPIVersion || l.Kind != listKind {
		return nil, fmt.Errorf("unsupported integration type schemas %s %s (supported: %s %s)", l.APIVersion, l.Kind, supportedAPIVersion, listKind)
	}

	s := &integrationschemas.Snapshot{Source: source}
	for _, it := range l.Items {
		if it.APIVersion != supportedAPIVersion || it.Kind != itemKind {
			return nil, fmt.Errorf("unsupported integration type schema item %s %s", it.APIVersion, it.Kind)
		}
		t, err := reduceType(it.Spec)
		if err != nil {
			return nil, err
		}
		if t.Version == "" {
			continue // no version that can be created, for example types that were only imported from Mimir
		}
		s.Types = append(s.Types, t)
	}
	sort.Slice(s.Types, func(i, j int) bool { return s.Types[i].APIType < s.Types[j].APIType })
	return s.Marshal()
}

func reduceType(t upstreamType) (integrationschemas.IntegrationType, error) {
	out := integrationschemas.IntegrationType{APIType: t.Type, Name: t.Name}
	for _, v := range t.Versions {
		if !v.CanCreate {
			continue
		}
		if out.Version != "" {
			return integrationschemas.IntegrationType{}, fmt.Errorf("integration type %q has more than one version that can be created (%s, %s); only one is supported", t.Type, out.Version, v.Version)
		}
		fields, err := reduceFields(t.Type+"/"+v.Version, v.Options)
		if err != nil {
			return integrationschemas.IntegrationType{}, err
		}
		out.Version, out.Fields = v.Version, fields
	}
	return out, nil
}

func reduceFields(path string, fields []upstreamField) ([]*integrationschemas.Field, error) {
	var out []*integrationschemas.Field
	for _, f := range fields {
		p := path + "." + f.PropertyName
		sf := &integrationschemas.Field{
			APIName:     f.PropertyName,
			Element:     f.Element,
			Description: f.Description,
			// A field that depends on another one is not always required.
			Required: f.Required && f.DependsOn == "" && f.ShowWhen.Field == "",
			Secure:   f.Secure,
		}
		if sf.Description == "" {
			sf.Description = f.Label
		}
		for _, o := range f.SelectOptions {
			value, err := selectValue(o)
			if err != nil {
				return nil, fmt.Errorf("field %s: %w", p, err)
			}
			sf.AllowedValues = append(sf.AllowedValues, value)
		}
		if len(f.SubformOptions) > 0 {
			sub, err := reduceFields(p, f.SubformOptions)
			if err != nil {
				return nil, err
			}
			sf.Children = sub
		}
		out = append(out, sf)
	}
	return out, nil
}

// selectValue returns the value of a select option as a string. Values the API reports as numbers
// are not supported yet, and fail instead of being converted.
func selectValue(o upstreamOption) (string, error) {
	if o.Value.String == nil {
		return "", fmt.Errorf("select option %q has a non-string value", o.Label)
	}
	return *o.Value.String, nil
}
