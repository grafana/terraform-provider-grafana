// equiv-delete-team removes the Grafana team used by equivalence tests (same as make equivalence-test-delete-team).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type searchResponse struct {
	Teams []struct {
		ID int64 `json:"id"`
	} `json:"teams"`
}

func firstTeamID(body []byte) (id int64, found bool, err error) {
	var r searchResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return 0, false, err
	}
	if len(r.Teams) == 0 {
		return 0, false, nil
	}
	return r.Teams[0].ID, true, nil
}

func splitBasicAuth(s string) (user, pass string) {
	i := strings.IndexByte(s, ':')
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i+1:]
}

func run() error {
	baseStr := strings.TrimSuffix(os.Getenv("GRAFANA_URL"), "/")
	if baseStr == "" {
		baseStr = "http://localhost:3000"
	}
	auth := os.Getenv("GRAFANA_AUTH")
	if auth == "" {
		auth = "admin:admin"
	}
	name := os.Getenv("EQUIV_TEAM_NAME")
	if name == "" {
		return fmt.Errorf("EQUIV_TEAM_NAME must be set")
	}

	baseURL, err := url.Parse(baseStr)
	if err != nil {
		return fmt.Errorf("GRAFANA_URL: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user, pass := splitBasicAuth(auth)

	searchURL := baseURL.ResolveReference(&url.URL{
		Path:     "/api/teams/search",
		RawQuery: url.Values{"name": []string{name}}.Encode(),
	}).String()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, pass)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("search teams at %s: %w", baseURL.Redacted(), err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("search teams: HTTP %s: %s", resp.Status, truncate(string(body), 400))
	}

	id, found, err := firstTeamID(body)
	if err != nil {
		return fmt.Errorf("parse search response: %w", err)
	}
	if !found {
		fmt.Printf("No team named %s found\n", name)
		return nil
	}

	deleteURL := baseURL.ResolveReference(&url.URL{
		Path: "/api/teams/" + strconv.FormatInt(id, 10),
	}).String()

	req2, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
	if err != nil {
		return err
	}
	req2.SetBasicAuth(user, pass)

	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		return fmt.Errorf("delete team %d: %w", id, err)
	}
	defer resp2.Body.Close()
	_, _ = io.Copy(io.Discard, resp2.Body)

	if resp2.StatusCode != http.StatusOK && resp2.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete team: HTTP %s", resp2.Status)
	}

	fmt.Printf("Deleted team id=%d (%s)\n", id, name)
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
