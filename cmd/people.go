package cmd

// `gws people get` wraps People.Get for programmatic consumers. The default
// ergonomic People surface stays under `gws contacts` — this command exists
// because the People API JSON shape (resourceName, personFields-driven
// fields like emailAddresses[].metadata.source, etc.) is what scripts want
// when speaking the API directly. Backed by --raw + --params per #188.

import (
	"context"
	"fmt"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/spf13/cobra"
	people "google.golang.org/api/people/v1"
)

var peopleCmd = &cobra.Command{
	Use:   "people",
	Short: "Direct People API access",
	Long: `Direct wrappers around the People API. Designed for programmatic
consumers — see also "gws contacts" for the ergonomic surface.`,
}

var peopleGetCmd = &cobra.Command{
	Use:   "get [resource-name]",
	Short: "Get a person by resourceName",
	Long: `Get a person by resourceName.

Either pass <resource-name> as a positional argument or supply it via --params.
Required parameter: personFields (defaults to the legacy default for
"gws contacts get" when omitted).

Examples:
  gws people get --params '{"resourceName":"people/me","personFields":"emailAddresses"}' --raw
  gws people get people/me --params '{"personFields":"names,emailAddresses"}'`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPeopleGet,
}

func init() {
	rootCmd.AddCommand(peopleCmd)
	peopleCmd.AddCommand(peopleGetCmd)
	addRawParamsFlags(peopleGetCmd)
	peopleGetCmd.Flags().String("person-fields", "", "Comma-separated personFields mask (overridden by --params personFields)")

	// Surface --raw + --params on the existing `gws contacts get` too so
	// callers that already use that command can opt into the raw shape.
	addRawParamsFlags(contactsGetCmd)
}

func runPeopleGet(cmd *cobra.Command, args []string) error {
	p := GetPrinter()

	// Pre-validate that we have a resourceName from positional or
	// --params before touching auth. Surfaces input errors immediately
	// instead of after OAuth/config failures.
	params, perr := parseParams(cmd)
	if perr != nil {
		return p.PrintError(perr)
	}
	hasResource := len(args) > 0 && args[0] != ""
	if v, ok := paramString(params, "resourceName"); ok && v != "" {
		hasResource = true
	}
	if !hasResource {
		return usageErrorf("people get: resourceName is required (positional arg or --params resourceName)")
	}

	ctx := context.Background()
	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}
	svc, err := factory.People()
	if err != nil {
		return p.PrintError(err)
	}
	return runPeopleGetWithSvc(cmd, svc, args)
}

// runPeopleGetWithSvc is the testable inner half of runPeopleGet — takes
// an injected *people.Service so the runner can be exercised against an
// httptest backend without needing real OAuth.
func runPeopleGetWithSvc(cmd *cobra.Command, svc *people.Service, args []string) error {
	p := GetPrinter()
	params, perr := parseParams(cmd)
	if perr != nil {
		return p.PrintError(perr)
	}

	resourceName := ""
	if len(args) > 0 {
		resourceName = args[0]
	}
	if v, ok := paramString(params, "resourceName"); ok && v != "" {
		resourceName = v
	}
	if resourceName == "" {
		return usageErrorf("people get: resourceName is required (positional arg or --params resourceName)")
	}

	pf, _ := cmd.Flags().GetString("person-fields")
	if v, ok := paramString(params, "personFields"); ok && v != "" {
		pf = v
	}
	if pf == "" {
		pf = personFields
	}

	call := svc.People.Get(resourceName).PersonFields(pf)
	if sources, ok := paramStringSlice(params, "sources"); ok && len(sources) > 0 {
		call = call.Sources(sources...)
	}
	if v, ok := paramString(params, "requestMask.includeField"); ok && v != "" {
		call = call.RequestMaskIncludeField(v)
	}

	person, err := call.Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get person: %w", err))
	}

	if isRaw(cmd) {
		return printRaw(person)
	}
	return p.Print(formatPerson(person))
}
