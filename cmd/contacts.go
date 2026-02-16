package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
	"google.golang.org/api/people/v1"
)

var contactsCmd = &cobra.Command{
	Use:   "contacts",
	Short: "Manage Google Contacts",
	Long:  "Commands for interacting with Google Contacts via the People API.",
}

var contactsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List contacts",
	Long:  "Lists contacts from your Google account.",
	RunE:  runContactsList,
}

var contactsSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search contacts",
	Long:  "Searches contacts by name, email, or phone number.",
	Args:  cobra.ExactArgs(1),
	RunE:  runContactsSearch,
}

var contactsGetCmd = &cobra.Command{
	Use:   "get <resource-name>",
	Short: "Get contact details",
	Long:  "Gets detailed information about a specific contact by resource name (e.g., people/c1234567890).",
	Args:  cobra.ExactArgs(1),
	RunE:  runContactsGet,
}

var contactsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new contact",
	Long:  "Creates a new contact with a name, email, and/or phone number.",
	RunE:  runContactsCreate,
}

var contactsDeleteCmd = &cobra.Command{
	Use:   "delete <resource-name>",
	Short: "Delete a contact",
	Long:  "Deletes a contact by resource name (e.g., people/c1234567890).",
	Args:  cobra.ExactArgs(1),
	RunE:  runContactsDelete,
}

func init() {
	rootCmd.AddCommand(contactsCmd)
	contactsCmd.AddCommand(contactsListCmd)
	contactsCmd.AddCommand(contactsSearchCmd)
	contactsCmd.AddCommand(contactsGetCmd)
	contactsCmd.AddCommand(contactsCreateCmd)
	contactsCmd.AddCommand(contactsDeleteCmd)

	// List flags
	contactsListCmd.Flags().Int64("max", 100, "Maximum number of contacts to return")
	contactsListCmd.Flags().String("query", "", "Filter contacts (applied client-side)")

	// Create flags
	contactsCreateCmd.Flags().String("name", "", "Contact name (required)")
	contactsCreateCmd.Flags().String("email", "", "Contact email address")
	contactsCreateCmd.Flags().String("phone", "", "Contact phone number")
	contactsCreateCmd.MarkFlagRequired("name")
}

const personFields = "names,emailAddresses,phoneNumbers,organizations"

func runContactsList(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.People()
	if err != nil {
		return p.PrintError(err)
	}

	maxResults, _ := cmd.Flags().GetInt64("max")

	var allContacts []*people.Person
	pageToken := ""
	pageSize := maxResults
	if pageSize > 1000 {
		pageSize = 1000
	}

	for {
		call := svc.People.Connections.List("people/me").
			PersonFields(personFields).
			PageSize(pageSize)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to list contacts: %w", err))
		}

		allContacts = append(allContacts, resp.Connections...)

		if resp.NextPageToken == "" || int64(len(allContacts)) >= maxResults {
			break
		}
		pageToken = resp.NextPageToken
	}

	// Trim to max
	if int64(len(allContacts)) > maxResults {
		allContacts = allContacts[:maxResults]
	}

	results := make([]map[string]interface{}, 0, len(allContacts))
	for _, person := range allContacts {
		results = append(results, formatPerson(person))
	}

	return p.Print(map[string]interface{}{
		"contacts": results,
		"count":    len(results),
	})
}

func runContactsSearch(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.People()
	if err != nil {
		return p.PrintError(err)
	}

	query := args[0]

	resp, err := svc.People.SearchContacts().
		Query(query).
		ReadMask(personFields).
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to search contacts: %w", err))
	}

	results := make([]map[string]interface{}, 0, len(resp.Results))
	for _, result := range resp.Results {
		if result.Person != nil {
			results = append(results, formatPerson(result.Person))
		}
	}

	return p.Print(map[string]interface{}{
		"contacts": results,
		"count":    len(results),
		"query":    query,
	})
}

func runContactsGet(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.People()
	if err != nil {
		return p.PrintError(err)
	}

	resourceName := args[0]

	person, err := svc.People.Get(resourceName).
		PersonFields(personFields).
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get contact: %w", err))
	}

	return p.Print(formatPerson(person))
}

func runContactsCreate(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.People()
	if err != nil {
		return p.PrintError(err)
	}

	name, _ := cmd.Flags().GetString("name")
	email, _ := cmd.Flags().GetString("email")
	phone, _ := cmd.Flags().GetString("phone")

	person := &people.Person{
		Names: []*people.Name{
			{UnstructuredName: name},
		},
	}

	if email != "" {
		person.EmailAddresses = []*people.EmailAddress{
			{Value: email},
		}
	}

	if phone != "" {
		person.PhoneNumbers = []*people.PhoneNumber{
			{Value: phone},
		}
	}

	created, err := svc.People.CreateContact(person).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create contact: %w", err))
	}

	result := formatPerson(created)
	result["status"] = "created"
	return p.Print(result)
}

func runContactsDelete(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.People()
	if err != nil {
		return p.PrintError(err)
	}

	resourceName := args[0]

	_, err = svc.People.DeleteContact(resourceName).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete contact: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":        "deleted",
		"resource_name": resourceName,
	})
}

// formatPerson converts a People API Person into a display map.
func formatPerson(person *people.Person) map[string]interface{} {
	result := map[string]interface{}{
		"resource_name": person.ResourceName,
	}

	if len(person.Names) > 0 {
		result["name"] = person.Names[0].DisplayName
	}

	if len(person.EmailAddresses) > 0 {
		emails := make([]string, len(person.EmailAddresses))
		for i, e := range person.EmailAddresses {
			emails[i] = e.Value
		}
		result["emails"] = emails
	}

	if len(person.PhoneNumbers) > 0 {
		phones := make([]string, len(person.PhoneNumbers))
		for i, ph := range person.PhoneNumbers {
			phones[i] = ph.Value
		}
		result["phones"] = phones
	}

	if len(person.Organizations) > 0 {
		org := person.Organizations[0]
		orgInfo := map[string]interface{}{}
		if org.Name != "" {
			orgInfo["name"] = org.Name
		}
		if org.Title != "" {
			orgInfo["title"] = org.Title
		}
		if len(orgInfo) > 0 {
			result["organization"] = orgInfo
		}
	}

	return result
}
