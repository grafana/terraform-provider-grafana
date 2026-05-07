# Cleanup Grafana objects created by equivalence test applies (fixed team name / user login).
# Uses GRAFANA_URL (default http://localhost:3000) and GRAFANA_AUTH (default admin:admin).
#
# From repo root (primary):  make equivalence-test-delete-team
#                            make equivalence-test-delete-user
#                            make equivalence-test-delete-fixtures
# Standalone:               make -f equivalence-tests/Makefile.delete-resources.mk equivalence-test-delete-fixtures

EQUIV_TEAM_NAME ?= terraform-equivalence-grafana-team
EQUIV_USER_LOGIN ?= terraform-equiv-grafana-user

.PHONY: equivalence-test-delete-team equivalence-test-delete-user equivalence-test-delete-fixtures

# Removes the fixed-name team so another equivalence apply can create it (avoids HTTP 409).
equivalence-test-delete-team:
	@base="$${GRAFANA_URL:-http://localhost:3000}"; base="$${base%/}"; resp=$$(curl -sfS -u "$${GRAFANA_AUTH:-admin:admin}" "$$base/api/teams/search?name=$(EQUIV_TEAM_NAME)") || { echo "Failed to search teams at $$base"; exit 1; }; id=$$(printf '%s' "$$resp" | python3 -c 'import json,sys; d=json.load(sys.stdin); t=d.get("teams") or []; print(t[0]["id"] if t else "")'); if [ -z "$$id" ]; then echo "No team named $(EQUIV_TEAM_NAME) found"; else curl -sfS -o /dev/null -u "$${GRAFANA_AUTH:-admin:admin}" -X DELETE "$$base/api/teams/$$id" && echo "Deleted team id=$$id ($(EQUIV_TEAM_NAME))"; fi

# Removes the fixed-login user so another equivalence apply can create it (avoids HTTP 409).
equivalence-test-delete-user:
	@base="$${GRAFANA_URL:-http://localhost:3000}"; base="$${base%/}"; resp=$$(curl -sfS -u "$${GRAFANA_AUTH:-admin:admin}" "$$base/api/users/search?query=$(EQUIV_USER_LOGIN)") || { echo "Failed to search users at $$base"; exit 1; }; id=$$(printf '%s' "$$resp" | python3 -c 'import json,sys; d=json.load(sys.stdin); want=sys.argv[1]; users=d.get("users") or []; m=[u for u in users if u.get("login")==want]; print(m[0]["id"] if m else "")' "$(EQUIV_USER_LOGIN)"); if [ -z "$$id" ]; then echo "No user with login $(EQUIV_USER_LOGIN) found"; else curl -sfS -o /dev/null -u "$${GRAFANA_AUTH:-admin:admin}" -X DELETE "$$base/api/admin/users/$$id" && echo "Deleted user id=$$id ($(EQUIV_USER_LOGIN))"; fi

equivalence-test-delete-fixtures: equivalence-test-delete-team equivalence-test-delete-user
	@echo "Equivalence test fixtures cleanup finished (team + user)."
