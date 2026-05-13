// teamsearchid reads a Grafana GET /api/teams/search JSON body from stdin
// and prints the numeric id of the first team in "teams", or nothing if empty.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type searchResponse struct {
	Teams []struct {
		ID int64 `json:"id"`
	} `json:"teams"`
}

func main() {
	body, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(1)
	}
	var r searchResponse
	if err := json.Unmarshal(body, &r); err != nil {
		os.Exit(1)
	}
	if len(r.Teams) == 0 {
		return
	}
	fmt.Print(r.Teams[0].ID)
}
