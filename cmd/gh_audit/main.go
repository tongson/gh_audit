package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)
import (
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var organization string = os.Getenv("GITHUB_ORG")
var token string = os.Getenv("GITHUB_TOKEN")

type Table struct {
	users map[int]map[string]string
	teams map[int][]string
}

func main() {
	var dotCSV string
	if len(os.Args) > 0 {
		dotCSV = os.Args[1]
	} else {
		log.Fatal("Failure getting path to CSV file to write to.")
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	client := github.NewClient(tc)
	teamsTable, err := fillTeamsTable(client)
	checkError("Failure fetching Github team information\n", err)
	usersTable, err := fillUsersTable(client)
	checkError("Failure fetching Github members\n", err)
	rawTable := &Table{
		users: usersTable,
		teams: teamsTable,
	}
	data := generateData(rawTable)
	err = csvWrite(dotCSV, data)
	checkError("Failure creating file\n", err)
}

func generateData(rawTable *Table) [][]string {
	/*
		Slice that contains the ordering of IDs.
		You can get the total number of members from len(keys).
	*/
	keys := make([]int, len(rawTable.users))
	i := 0
	for key, _ := range rawTable.users {
		// fmt.Printf("%d:%d\n", i, key)
		keys[i] = key
		i++
	}
	sort.Ints(keys)
	data := make([][]string, len(keys)+1)
	data[0] = append(data[0], "ID")
	data[0] = append(data[0], "Login")
	data[0] = append(data[0], "Name")
	data[0] = append(data[0], "Type")
	data[0] = append(data[0], "Teams")
	ci := 1
	for _, y := range keys {
		id := strconv.Itoa(y)
		data[ci] = append(data[ci], id)
		data[ci] = append(data[ci], rawTable.users[y]["Login"])
		if rawTable.users[y]["Name"] != "" {
			data[ci] = append(data[ci], rawTable.users[y]["Name"])
		} else {
			data[ci] = append(data[ci], "")
		}
		data[ci] = append(data[ci], rawTable.users[y]["Type"])
		data[ci] = append(data[ci], strings.Join(rawTable.teams[y], ","))
		ci++
	}
	//ci should be equal to (len(keys)+1)
	return data
}

func csvWrite(dotCSV string, data [][]string) error {
	file, err := os.Create(dotCSV)
	if err != nil {
		fmt.Errorf("Error: %v\n", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	for _, value := range data {
		err := writer.Write(value)
		if err != nil {
			fmt.Errorf("Error: %v\n", err)
		}
	}
	defer writer.Flush()
	return err
}

func checkError(message string, err error) {
	if err != nil {
		log.Fatal(message, err)
	}
}

func fillTeamsTable(client *github.Client) (map[int][]string, error) {
	ListTeamMembersOpt := &github.OrganizationListTeamMembersOptions{
		ListOptions: github.ListOptions{PerPage: 30},
	}
	ListTeamsOpt := &github.ListOptions{PerPage: 30}

	/*
		Slice of all teams in the organization.
		tempTeamsTable = {
			0 = *github.Team
			...
		}
	*/
	tempTeamsTable := make([]*github.Team, 0)
	for {
		teams, resp, err := client.Organizations.ListTeams(organization, ListTeamsOpt)
		if err != nil {
			return nil, fmt.Errorf("Error: %v\n", err)
		}
		tempTeamsTable = append(tempTeamsTable, teams...)
		ListTeamsOpt.Page = resp.NextPage
		if resp.NextPage == 0 {
			break
		}
	}

	/*
		Map of teams. Each team has a slice for each member.
		tempTable = {
			*team.Name = {
				0 = *github.User
				...
			}
		}
	*/
	tempTable := make(map[string][]*github.User)
	for _, team := range tempTeamsTable {
		for {
			allTeamMembers, resp, err := client.Organizations.ListTeamMembers(*team.ID, ListTeamMembersOpt)
			if err != nil {
				return nil, fmt.Errorf("Error: %v\n", err)
			}
			tempTable[*team.Name] = append(tempTable[*team.Name], allTeamMembers...)
			ListTeamMembersOpt.ListOptions.Page = resp.NextPage
			if resp.NextPage == 0 {
				break
			}
		}
	}

	/*
		Map of member IDs. Each member ID has a slice of team names.
		teamTable = {
			*member.ID = {
				0 = *team.Name,
				...
			}
		}
	*/
	teamTable := make(map[int][]string)
	for team, members := range tempTable {
		for _, member := range members {
			teamTable[*member.ID] = append(teamTable[*member.ID], team)
		}
	}
	return teamTable, nil

}

func fillUsersTable(client *github.Client) (map[int]map[string]string, error) {
	ListMembersOpt := &github.ListMembersOptions{
		ListOptions: github.ListOptions{PerPage: 30},
	}

	var allMembers []*github.User
	for {
		members, resp, err := client.Organizations.ListMembers(organization, ListMembersOpt)
		if err != nil {
			return nil, fmt.Errorf("Error: %v\n", err)
		}
		allMembers = append(allMembers, members...)
		ListMembersOpt.ListOptions.Page = resp.NextPage
		if resp.NextPage == 0 {
			break
		}
	}

	/*
		Map of members. Each member has a map of k,v entries.
		usersTable = {
			*member.ID = {
				"Login" = *member.Login,
				"Type"  = *member.Type,
				"Name"  = *member.Name,
			}
		}
	*/
	usersTable := make(map[int]map[string]string)
	for _, member := range allMembers {
		usersTable[*member.ID] = map[string]string{"Login": *member.Login, "Type": *member.Type}
		user, _, err := client.Users.GetByID(*member.ID)
		if err != nil {
			return nil, fmt.Errorf("Error: %v\n", err)
		}
		if user.Name != nil {
			usersTable[*member.ID]["Name"] = *user.Name
		}

	}
	return usersTable, nil
}
