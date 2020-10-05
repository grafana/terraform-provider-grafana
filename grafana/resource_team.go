package grafana

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	gapi "github.com/nytm/go-grafana-api"
)

type TeamMember struct {
	ID    int64
	Email string
}

type MemberChange struct {
	Type   ChangeMemberType
	Member TeamMember
}

type ChangeMemberType int8

const (
	AddMember ChangeMemberType = iota
	RemoveMember
)

func ResourceTeam() *schema.Resource {
	return &schema.Resource{
		Create: CreateTeam,
		Read:   ReadTeam,
		Update: UpdateTeam,
		Delete: DeleteTeam,
		Exists: ExistsTeam,
		Importer: &schema.ResourceImporter{
			State: ImportTeam,
		},

		Schema: map[string]*schema.Schema{
			"team_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"email": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"members": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func CreateTeam(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	name := d.Get("name").(string)
	email := d.Get("email").(string)
	teamID, err := client.AddTeam(name, email)
	if err != nil && err.Error() == "409 Conflict" {
		return errors.New(fmt.Sprintf("Error: A Grafana Team with the name '%s' already exists.", name))
	}
	if err != nil {
		return err
	}

	d.SetId(strconv.FormatInt(teamID, 10))
	return UpdateMembers(d, meta)
}

func ReadTeam(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	teamID, _ := strconv.ParseInt(d.Id(), 10, 64)
	resp, err := client.Team(teamID)
	if err != nil && err.Error() == "404 Not Found" {
		log.Printf("[WARN] removing team %s from state because it no longer exists in grafana", d.Id())
		d.SetId("")
		return nil
	}
	if err != nil {
		return err
	}
	d.Set("name", resp.Name)
	d.Set("email", resp.Email)
	if err := ReadMembers(d, meta); err != nil {
		return err
	}
	return nil
}

func UpdateTeam(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	teamID, _ := strconv.ParseInt(d.Id(), 10, 64)
	if d.HasChange("name") || d.HasChange("email") {
		name := d.Get("name").(string)
		email := d.Get("email").(string)
		err := client.UpdateTeam(teamID, name, email)
		if err != nil {
			return err
		}
	}
	return UpdateMembers(d, meta)
}

func DeleteTeam(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	teamID, _ := strconv.ParseInt(d.Id(), 10, 64)
	return client.DeleteTeam(teamID)
}

func ExistsTeam(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*gapi.Client)
	teamID, _ := strconv.ParseInt(d.Id(), 10, 64)
	_, err := client.Team(teamID)
	if err != nil && err.Error() == "404 Not Found" {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, err
}

func ImportTeam(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	exists, err := ExistsTeam(d, meta)
	if err != nil || !exists {
		return nil, errors.New(fmt.Sprintf("Error: Unable to import Grafana Team: %s.", err))
	}
	err = ReadTeam(d, meta)
	if err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func ReadMembers(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	teamID, _ := strconv.ParseInt(d.Id(), 10, 64)
	teamMembers, err := client.TeamMembers(teamID)
	if err != nil {
		return err
	}
	memberSlice := []string{}
	for _, teamMember := range teamMembers {
		memberSlice = append(memberSlice, teamMember.Login)
	}
	d.Set("members", memberSlice)

	return nil
}

func UpdateMembers(d *schema.ResourceData, meta interface{}) error {
	stateMembers, configMembers, err := collectMembers(d)
	if err != nil {
		return err
	}
	//compile the list of differences between current state and config
	changes := memberChanges(stateMembers, configMembers)
	//retrieves the corresponding user IDs based on the email provided
	changes, err = addMemberIdsToChanges(meta, changes)
	if err != nil {
		return err
	}
	teamID, _ := strconv.ParseInt(d.Id(), 10, 64)
	//now we can make the corresponding updates so current state matches config
	return applyMemberChanges(meta, teamID, changes)
}

func collectMembers(d *schema.ResourceData) (map[string]TeamMember, map[string]TeamMember, error) {
	stateMembers, configMembers := make(map[string]TeamMember), make(map[string]TeamMember)

	// Get the lists of team members read in from Grafana state (old) and configured (new)
	state, config := d.GetChange("members")
	for _, u := range state.([]interface{}) {
		login := u.(string)
		// Sanity check that a member isn't specified twice within a team
		if _, ok := stateMembers[login]; ok {
			return nil, nil, errors.New(fmt.Sprintf("Error: Team Member '%s' cannot be specified multiple times.", login))
		}
		stateMembers[login] = TeamMember{0, login}
	}
	for _, u := range config.([]interface{}) {
		login := u.(string)
		// Sanity check that a member isn't specified twice within a team
		if _, ok := configMembers[login]; ok {
			return nil, nil, errors.New(fmt.Sprintf("Error: Team Member '%s' cannot be specified multiple times.", login))
		}
		configMembers[login] = TeamMember{0, login}
	}

	return stateMembers, configMembers, nil
}

func memberChanges(stateMembers, configMembers map[string]TeamMember) []MemberChange {
	var changes []MemberChange
	for _, user := range configMembers {
		_, ok := stateMembers[user.Email]
		if !ok {
			// Member doesn't exist in Grafana's state for the team, should be added.
			changes = append(changes, MemberChange{AddMember, user})
			continue
		}
	}
	for _, user := range stateMembers {
		if _, ok := configMembers[user.Email]; !ok {
			// Member exists in Grafana's state for the team, but isn't
			// present in the team configuration, should be removed.
			changes = append(changes, MemberChange{RemoveMember, user})
		}
	}
	return changes
}

func addMemberIdsToChanges(meta interface{}, changes []MemberChange) ([]MemberChange, error) {
	client := meta.(*gapi.Client)
	gUserMap := make(map[string]int64)
	gUsers, err := client.Users()
	if err != nil {
		return nil, err
	}
	for _, u := range gUsers {
		gUserMap[u.Email] = u.Id
	}
	var output []MemberChange

	for _, change := range changes {
		id, ok := gUserMap[change.Member.Email]
		if !ok {
			return nil, errors.New(fmt.Sprintf("Error adding user %s. User does not exist in Grafana.", change.Member.Email))
		}

		change.Member.ID = id
		output = append(output, change)
	}
	return output, nil
}

func applyMemberChanges(meta interface{}, teamId int64, changes []MemberChange) error {
	var err error
	client := meta.(*gapi.Client)
	for _, change := range changes {
		u := change.Member
		switch change.Type {
		case AddMember:
			err = client.AddTeamMember(teamId, u.ID)
		case RemoveMember:
			err = client.RemoveMemberFromTeam(teamId, u.ID)
		}
		if err != nil && err.Error() != "409 Conflict" {
			return err
		}
	}
	return nil
}
