package grafana

import (
	"strings"
	"testing"
)

func TestValidateExplicitRoleVersionDiff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		isCreate       bool
		versionChanged bool
		oldVer         int
		newVer         int
		wantErr        string
	}{
		{
			name:           "create explicit version 1",
			isCreate:       true,
			versionChanged: true,
			newVer:         1,
		},
		{
			name:           "create explicit version not 1",
			isCreate:       true,
			versionChanged: true,
			newVer:         2,
			wantErr:        "version must be 1",
		},
		{
			name:           "update no version change",
			isCreate:       false,
			versionChanged: false,
			oldVer:         3,
			newVer:         3,
		},
		{
			name:           "update increment by 1",
			isCreate:       false,
			versionChanged: true,
			oldVer:         4,
			newVer:         5,
		},
		{
			name:           "update skip version",
			isCreate:       false,
			versionChanged: true,
			oldVer:         5,
			newVer:         7,
			wantErr:        "increase by exactly 1",
		},
		{
			name:           "update same version when changed flag set but equal values edge",
			isCreate:       false,
			versionChanged: true,
			oldVer:         2,
			newVer:         2,
			wantErr:        "increase by exactly 1",
		},
		{
			name:           "auto to explicit would be old 5 new 7",
			isCreate:       false,
			versionChanged: true,
			oldVer:         5,
			newVer:         7,
			wantErr:        "expected 6, got 7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateExplicitRoleVersionDiff(tt.isCreate, tt.versionChanged, tt.oldVer, tt.newVer)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q should contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}
