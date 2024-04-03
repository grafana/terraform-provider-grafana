package main

import (
	"encoding/json"
	"fmt"
)

type state map[string]interface{}
type resource map[string]interface{}

func (s state) resources() []resource {
	values := s["values"].(map[string]interface{})
	rootModule := values["root_module"].(map[string]interface{})
	var resources []resource
	for _, resourceInterface := range rootModule["resources"].([]interface{}) {
		resources = append(resources, resourceInterface.(map[string]interface{}))
	}
	return resources
}

func (s state) getResource(resourceType string, name string) (resource, error) {
	for _, resource := range s.resources() {
		if resource.resourceType() == resourceType && resource.name() == name {
			return resource, nil
		}
	}
	return nil, fmt.Errorf("resource not found")
}

func (r resource) name() string {
	return r["name"].(string)
}

func (r resource) resourceType() string {
	return r["type"].(string)
}

func (r resource) values() map[string]interface{} {
	return r["values"].(map[string]interface{})
}

func getState(dir string) (state, error) {
	state, err := runTerraformWithOutput(dir, "show", "-json")
	if err != nil {
		return nil, fmt.Errorf("failed to read terraform state: %w", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(state, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse terraform state: %w", err)
	}
	return parsed, nil
}
