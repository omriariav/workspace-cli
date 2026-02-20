package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
)

var groupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "Manage Google Groups",
	Long:  "Commands for interacting with Google Groups via the Admin Directory API.",
}

var groupsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List groups",
	Long:  "Lists Google Groups in your domain.",
	RunE:  runGroupsList,
}

var groupsMembersCmd = &cobra.Command{
	Use:   "members <group-email>",
	Short: "List group members",
	Long:  "Lists members of a Google Group by group email address.",
	Args:  cobra.ExactArgs(1),
	RunE:  runGroupsMembers,
}

func init() {
	rootCmd.AddCommand(groupsCmd)
	groupsCmd.AddCommand(groupsListCmd)
	groupsCmd.AddCommand(groupsMembersCmd)

	// List flags
	groupsListCmd.Flags().Int64("max", 50, "Maximum number of groups to return")
	groupsListCmd.Flags().String("domain", "", "Filter by domain")
	groupsListCmd.Flags().String("user-email", "", "Filter groups for a specific user")

	// Members flags
	groupsMembersCmd.Flags().Int64("max", 50, "Maximum number of members to return")
	groupsMembersCmd.Flags().String("role", "", "Filter by role (OWNER, MANAGER, MEMBER)")
}

func runGroupsList(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Admin()
	if err != nil {
		return p.PrintError(err)
	}

	maxResults, _ := cmd.Flags().GetInt64("max")
	domain, _ := cmd.Flags().GetString("domain")
	userEmail, _ := cmd.Flags().GetString("user-email")

	if domain != "" && userEmail != "" {
		return p.PrintError(fmt.Errorf("--domain and --user-email are mutually exclusive"))
	}

	call := svc.Groups.List()

	if domain != "" {
		call = call.Domain(domain)
	} else if userEmail != "" {
		call = call.UserKey(userEmail)
	} else {
		call = call.Customer("my_customer")
	}
	call = call.MaxResults(maxResults)

	resp, err := call.Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list groups: %w", err))
	}

	groups := make([]map[string]interface{}, 0, len(resp.Groups))
	for _, g := range resp.Groups {
		group := map[string]interface{}{
			"id":    g.Id,
			"email": g.Email,
			"name":  g.Name,
		}
		if g.Description != "" {
			group["description"] = g.Description
		}
		if g.DirectMembersCount != 0 {
			group["member_count"] = g.DirectMembersCount
		}
		groups = append(groups, group)
	}

	return p.Print(map[string]interface{}{
		"groups": groups,
		"count":  len(groups),
	})
}

func runGroupsMembers(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Admin()
	if err != nil {
		return p.PrintError(err)
	}

	groupKey := args[0]
	maxResults, _ := cmd.Flags().GetInt64("max")
	role, _ := cmd.Flags().GetString("role")

	call := svc.Members.List(groupKey).MaxResults(maxResults)
	if role != "" {
		call = call.Roles(role)
	}

	resp, err := call.Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to list members: %w", err))
	}

	members := make([]map[string]interface{}, 0, len(resp.Members))
	for _, m := range resp.Members {
		member := map[string]interface{}{
			"id":    m.Id,
			"email": m.Email,
			"role":  m.Role,
			"type":  m.Type,
		}
		if m.Status != "" {
			member["status"] = m.Status
		}
		members = append(members, member)
	}

	return p.Print(map[string]interface{}{
		"group":   groupKey,
		"members": members,
		"count":   len(members),
	})
}
