package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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

var contactsUpdateCmd = &cobra.Command{
	Use:   "update <resource-name>",
	Short: "Update a contact",
	Long:  "Updates an existing contact by resource name. Specify fields to update via flags.",
	Args:  cobra.ExactArgs(1),
	RunE:  runContactsUpdate,
}

var contactsBatchCreateCmd = &cobra.Command{
	Use:   "batch-create",
	Short: "Batch create contacts",
	Long:  "Creates multiple contacts from a JSON file. The file should contain an array of contact objects.",
	RunE:  runContactsBatchCreate,
}

var contactsBatchUpdateCmd = &cobra.Command{
	Use:   "batch-update",
	Short: "Batch update contacts",
	Long:  "Updates multiple contacts from a JSON file. The file should contain a map of resource names to contact objects.",
	RunE:  runContactsBatchUpdate,
}

var contactsBatchDeleteCmd = &cobra.Command{
	Use:   "batch-delete",
	Short: "Batch delete contacts",
	Long:  "Deletes multiple contacts by resource names.",
	RunE:  runContactsBatchDelete,
}

var contactsDirectoryCmd = &cobra.Command{
	Use:   "directory",
	Short: "List directory people",
	Long:  "Lists people in the organization's directory. Requires directory.readonly scope.",
	RunE:  runContactsDirectory,
}

var contactsDirectorySearchCmd = &cobra.Command{
	Use:   "directory-search",
	Short: "Search directory people",
	Long:  "Searches people in the organization's directory by query. Requires directory.readonly scope.",
	RunE:  runContactsDirectorySearch,
}

var contactsPhotoCmd = &cobra.Command{
	Use:   "photo <resource-name>",
	Short: "Update contact photo",
	Long:  "Updates a contact's photo from an image file (JPEG or PNG).",
	Args:  cobra.ExactArgs(1),
	RunE:  runContactsPhoto,
}

var contactsDeletePhotoCmd = &cobra.Command{
	Use:   "delete-photo <resource-name>",
	Short: "Delete contact photo",
	Long:  "Deletes a contact's photo by resource name.",
	Args:  cobra.ExactArgs(1),
	RunE:  runContactsDeletePhoto,
}

var contactsResolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Resolve multiple contacts",
	Long:  "Gets multiple contacts by their resource names in a single batch request.",
	RunE:  runContactsResolve,
}

func init() {
	rootCmd.AddCommand(contactsCmd)
	contactsCmd.AddCommand(contactsListCmd)
	contactsCmd.AddCommand(contactsSearchCmd)
	contactsCmd.AddCommand(contactsGetCmd)
	contactsCmd.AddCommand(contactsCreateCmd)
	contactsCmd.AddCommand(contactsDeleteCmd)
	contactsCmd.AddCommand(contactsUpdateCmd)
	contactsCmd.AddCommand(contactsBatchCreateCmd)
	contactsCmd.AddCommand(contactsBatchUpdateCmd)
	contactsCmd.AddCommand(contactsBatchDeleteCmd)
	contactsCmd.AddCommand(contactsDirectoryCmd)
	contactsCmd.AddCommand(contactsDirectorySearchCmd)
	contactsCmd.AddCommand(contactsPhotoCmd)
	contactsCmd.AddCommand(contactsDeletePhotoCmd)
	contactsCmd.AddCommand(contactsResolveCmd)

	// List flags
	contactsListCmd.Flags().Int64("max", 50, "Maximum number of contacts to return")

	// Create flags
	contactsCreateCmd.Flags().String("name", "", "Contact name (required)")
	contactsCreateCmd.Flags().String("email", "", "Contact email address")
	contactsCreateCmd.Flags().String("phone", "", "Contact phone number")
	contactsCreateCmd.MarkFlagRequired("name")

	// Update flags
	contactsUpdateCmd.Flags().String("name", "", "Updated contact name")
	contactsUpdateCmd.Flags().String("email", "", "Updated email address")
	contactsUpdateCmd.Flags().String("phone", "", "Updated phone number")
	contactsUpdateCmd.Flags().String("organization", "", "Updated organization name")
	contactsUpdateCmd.Flags().String("title", "", "Updated job title")
	contactsUpdateCmd.Flags().String("etag", "", "Etag for concurrency control (from get command)")

	// Batch create flags
	contactsBatchCreateCmd.Flags().String("file", "", "Path to JSON file with contacts array (required)")
	contactsBatchCreateCmd.MarkFlagRequired("file")

	// Batch update flags
	contactsBatchUpdateCmd.Flags().String("file", "", "Path to JSON file with contacts map (required)")
	contactsBatchUpdateCmd.MarkFlagRequired("file")

	// Batch delete flags
	contactsBatchDeleteCmd.Flags().StringArray("resources", nil, "Resource names to delete (repeatable, e.g. --resources people/c1 --resources people/c2)")
	contactsBatchDeleteCmd.MarkFlagRequired("resources")

	// Directory flags
	contactsDirectoryCmd.Flags().Int64("max", 50, "Maximum number of directory people to return")

	// Directory search flags
	contactsDirectorySearchCmd.Flags().String("query", "", "Search query (required)")
	contactsDirectorySearchCmd.Flags().Int64("max", 50, "Maximum number of results to return")
	contactsDirectorySearchCmd.MarkFlagRequired("query")

	// Photo flags
	contactsPhotoCmd.Flags().String("file", "", "Path to image file, JPEG or PNG (required)")
	contactsPhotoCmd.MarkFlagRequired("file")

	// Resolve flags
	contactsResolveCmd.Flags().StringArray("ids", nil, "Resource names to resolve (repeatable, e.g. --ids people/c1 --ids people/c2)")
	contactsResolveCmd.MarkFlagRequired("ids")
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

func runContactsUpdate(cmd *cobra.Command, args []string) error {
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

	name, _ := cmd.Flags().GetString("name")
	email, _ := cmd.Flags().GetString("email")
	phone, _ := cmd.Flags().GetString("phone")
	organization, _ := cmd.Flags().GetString("organization")
	title, _ := cmd.Flags().GetString("title")
	etag, _ := cmd.Flags().GetString("etag")

	// Determine which fields the user wants to update
	var updateFields []string
	if cmd.Flags().Changed("name") {
		updateFields = append(updateFields, "names")
	}
	if cmd.Flags().Changed("email") {
		updateFields = append(updateFields, "emailAddresses")
	}
	if cmd.Flags().Changed("phone") {
		updateFields = append(updateFields, "phoneNumbers")
	}
	if cmd.Flags().Changed("organization") || cmd.Flags().Changed("title") {
		updateFields = append(updateFields, "organizations")
	}

	if len(updateFields) == 0 {
		return p.PrintError(fmt.Errorf("at least one field to update must be specified (--name, --email, --phone, --organization, --title)"))
	}

	// Fetch existing contact to get source metadata required by the API
	existing, err := svc.People.Get(resourceName).
		PersonFields(strings.Join(updateFields, ",")).
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to fetch contact for update: %w", err))
	}

	// Use the existing person as base and merge updates
	person := existing

	if etag != "" {
		person.Etag = etag
	}

	if cmd.Flags().Changed("name") {
		if name != "" {
			person.Names = []*people.Name{{UnstructuredName: name}}
		} else {
			person.Names = nil
		}
	}

	if cmd.Flags().Changed("email") {
		if email != "" {
			person.EmailAddresses = []*people.EmailAddress{{Value: email}}
		} else {
			person.EmailAddresses = nil
		}
	}

	if cmd.Flags().Changed("phone") {
		if phone != "" {
			person.PhoneNumbers = []*people.PhoneNumber{{Value: phone}}
		} else {
			person.PhoneNumbers = nil
		}
	}

	if cmd.Flags().Changed("organization") || cmd.Flags().Changed("title") {
		org := &people.Organization{}
		if len(person.Organizations) > 0 {
			org = person.Organizations[0]
		}
		if cmd.Flags().Changed("organization") {
			org.Name = organization
		}
		if cmd.Flags().Changed("title") {
			org.Title = title
		}
		person.Organizations = []*people.Organization{org}
	}

	updateMask := strings.Join(updateFields, ",")

	updated, err := svc.People.UpdateContact(resourceName, person).
		UpdatePersonFields(updateMask).
		PersonFields(personFields).
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update contact: %w", err))
	}

	result := formatPerson(updated)
	result["status"] = "updated"
	return p.Print(result)
}

func runContactsBatchCreate(cmd *cobra.Command, args []string) error {
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

	filePath, _ := cmd.Flags().GetString("file")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to read file %s: %w", filePath, err))
	}

	var contacts []*people.Person
	if err := json.Unmarshal(data, &contacts); err != nil {
		return p.PrintError(fmt.Errorf("failed to parse JSON file: %w", err))
	}

	if len(contacts) == 0 {
		return p.PrintError(fmt.Errorf("no contacts found in file"))
	}

	contactsToCreate := make([]*people.ContactToCreate, len(contacts))
	for i, c := range contacts {
		contactsToCreate[i] = &people.ContactToCreate{
			ContactPerson: c,
		}
	}

	req := &people.BatchCreateContactsRequest{
		Contacts: contactsToCreate,
		ReadMask: personFields,
	}

	resp, err := svc.People.BatchCreateContacts(req).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to batch create contacts: %w", err))
	}

	results := make([]map[string]interface{}, 0, len(resp.CreatedPeople))
	var failed []map[string]interface{}
	for i, pr := range resp.CreatedPeople {
		if pr.Person != nil {
			results = append(results, formatPerson(pr.Person))
		} else {
			entry := map[string]interface{}{"index": i}
			if pr.Status != nil {
				entry["code"] = pr.Status.Code
				entry["message"] = pr.Status.Message
			}
			failed = append(failed, entry)
		}
	}

	out := map[string]interface{}{
		"status":   "created",
		"contacts": results,
		"count":    len(results),
	}
	if len(failed) > 0 {
		out["failed"] = failed
		out["failed_count"] = len(failed)
	}
	return p.Print(out)
}

func runContactsBatchUpdate(cmd *cobra.Command, args []string) error {
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

	filePath, _ := cmd.Flags().GetString("file")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to read file %s: %w", filePath, err))
	}

	// The file format: {"contacts": {"people/c123": {...}, ...}, "update_mask": "names,emailAddresses"}
	var fileData struct {
		Contacts   map[string]people.Person `json:"contacts"`
		UpdateMask string                   `json:"update_mask"`
	}
	if err := json.Unmarshal(data, &fileData); err != nil {
		return p.PrintError(fmt.Errorf("failed to parse JSON file: %w", err))
	}

	if len(fileData.Contacts) == 0 {
		return p.PrintError(fmt.Errorf("no contacts found in file"))
	}

	if fileData.UpdateMask == "" {
		return p.PrintError(fmt.Errorf("update_mask is required in the JSON file"))
	}

	req := &people.BatchUpdateContactsRequest{
		Contacts:   fileData.Contacts,
		UpdateMask: fileData.UpdateMask,
		ReadMask:   personFields,
	}

	resp, err := svc.People.BatchUpdateContacts(req).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to batch update contacts: %w", err))
	}

	results := make(map[string]interface{}, len(resp.UpdateResult))
	var failed []map[string]interface{}
	for resourceName, pr := range resp.UpdateResult {
		if pr.Person != nil {
			results[resourceName] = formatPerson(pr.Person)
		} else {
			entry := map[string]interface{}{"resource_name": resourceName}
			if pr.Status != nil {
				entry["code"] = pr.Status.Code
				entry["message"] = pr.Status.Message
			}
			failed = append(failed, entry)
		}
	}

	out := map[string]interface{}{
		"status":  "updated",
		"results": results,
		"count":   len(results),
	}
	if len(failed) > 0 {
		out["failed"] = failed
		out["failed_count"] = len(failed)
	}
	return p.Print(out)
}

func runContactsBatchDelete(cmd *cobra.Command, args []string) error {
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

	resources, _ := cmd.Flags().GetStringArray("resources")

	req := &people.BatchDeleteContactsRequest{
		ResourceNames: resources,
	}

	_, err = svc.People.BatchDeleteContacts(req).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to batch delete contacts: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":         "deleted",
		"resource_names": resources,
		"count":          len(resources),
	})
}

func runContactsDirectory(cmd *cobra.Command, args []string) error {
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

	var allPeople []*people.Person
	pageToken := ""
	pageSize := maxResults
	if pageSize > 1000 {
		pageSize = 1000
	}

	for {
		call := svc.People.ListDirectoryPeople().
			ReadMask(personFields).
			Sources("DIRECTORY_SOURCE_TYPE_DOMAIN_PROFILE").
			PageSize(pageSize)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to list directory people: %w", err))
		}

		allPeople = append(allPeople, resp.People...)

		if resp.NextPageToken == "" || int64(len(allPeople)) >= maxResults {
			break
		}
		pageToken = resp.NextPageToken
	}

	// Trim to max
	if int64(len(allPeople)) > maxResults {
		allPeople = allPeople[:maxResults]
	}

	results := make([]map[string]interface{}, 0, len(allPeople))
	for _, person := range allPeople {
		results = append(results, formatPerson(person))
	}

	return p.Print(map[string]interface{}{
		"contacts": results,
		"count":    len(results),
	})
}

func runContactsDirectorySearch(cmd *cobra.Command, args []string) error {
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

	query, _ := cmd.Flags().GetString("query")
	maxResults, _ := cmd.Flags().GetInt64("max")

	var allPeople []*people.Person
	pageToken := ""
	pageSize := maxResults
	if pageSize > 500 {
		pageSize = 500 // API max for SearchDirectoryPeople
	}

	for {
		call := svc.People.SearchDirectoryPeople().
			Query(query).
			ReadMask(personFields).
			Sources("DIRECTORY_SOURCE_TYPE_DOMAIN_PROFILE").
			PageSize(pageSize)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to search directory people: %w", err))
		}

		allPeople = append(allPeople, resp.People...)

		if resp.NextPageToken == "" || int64(len(allPeople)) >= maxResults {
			break
		}
		pageToken = resp.NextPageToken
	}

	// Trim to max
	if int64(len(allPeople)) > maxResults {
		allPeople = allPeople[:maxResults]
	}

	results := make([]map[string]interface{}, 0, len(allPeople))
	for _, person := range allPeople {
		results = append(results, formatPerson(person))
	}

	return p.Print(map[string]interface{}{
		"contacts": results,
		"count":    len(results),
		"query":    query,
	})
}

func runContactsPhoto(cmd *cobra.Command, args []string) error {
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
	filePath, _ := cmd.Flags().GetString("file")

	photoData, err := os.ReadFile(filePath)
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to read photo file %s: %w", filePath, err))
	}

	encodedPhoto := base64.StdEncoding.EncodeToString(photoData)

	req := &people.UpdateContactPhotoRequest{
		PhotoBytes:   encodedPhoto,
		PersonFields: personFields,
	}

	resp, err := svc.People.UpdateContactPhoto(resourceName, req).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to update contact photo: %w", err))
	}

	result := map[string]interface{}{
		"status":        "photo_updated",
		"resource_name": resourceName,
	}
	if resp.Person != nil {
		result = formatPerson(resp.Person)
		result["status"] = "photo_updated"
	}

	return p.Print(result)
}

func runContactsDeletePhoto(cmd *cobra.Command, args []string) error {
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

	_, err = svc.People.DeleteContactPhoto(resourceName).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete contact photo: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":        "photo_deleted",
		"resource_name": resourceName,
	})
}

func runContactsResolve(cmd *cobra.Command, args []string) error {
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

	ids, _ := cmd.Flags().GetStringArray("ids")

	resp, err := svc.People.GetBatchGet().
		ResourceNames(ids...).
		PersonFields(personFields).
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to resolve contacts: %w", err))
	}

	results := make([]map[string]interface{}, 0, len(resp.Responses))
	var notFound []string
	var errors []map[string]interface{}
	for _, pr := range resp.Responses {
		if pr.Person != nil {
			results = append(results, formatPerson(pr.Person))
		} else if pr.Status != nil && pr.Status.Code == 5 {
			// Code 5 = NOT_FOUND
			notFound = append(notFound, pr.RequestedResourceName)
		} else {
			entry := map[string]interface{}{"resource_name": pr.RequestedResourceName}
			if pr.Status != nil {
				entry["code"] = pr.Status.Code
				entry["message"] = pr.Status.Message
			}
			errors = append(errors, entry)
		}
	}

	out := map[string]interface{}{"contacts": results, "count": len(results)}
	if len(notFound) > 0 {
		out["not_found"] = notFound
	}
	if len(errors) > 0 {
		out["errors"] = errors
	}
	return p.Print(out)
}

// formatPerson converts a People API Person into a display map.
func formatPerson(person *people.Person) map[string]interface{} {
	result := map[string]interface{}{
		"resource_name": person.ResourceName,
	}

	if person.Etag != "" {
		result["etag"] = person.Etag
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
