package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/t12e/redan-plugin/internal/auth"
	"github.com/t12e/redan-plugin/internal/client"
	"github.com/t12e/redan-plugin/internal/models"
)

var version = "0.2.0"

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, out io.Writer) error {
	jsonOutput, args := takeJSON(args)
	if len(args) == 0 || args[0] == "help" {
		return usage(out)
	}
	if args[0] == "version" {
		if jsonOutput {
			return writeJSON(out, map[string]string{"version": version})
		}
		_, err := fmt.Fprintln(out, version)
		return err
	}
	if args[0] == "auth" {
		return authentication(ctx, args[1:], jsonOutput, out)
	}
	if len(args) < 2 {
		return errors.New("usage: redan [--json] faq categories|faqs <command> [options]")
	}
	api, err := client.NewAuthenticated()
	if err != nil {
		return err
	}
	switch args[0] {
	case "faq":
		return faq(ctx, api, args[1:], jsonOutput, out)
	case "forms":
		return forms(ctx, api, args[1:], jsonOutput, out)
	default:
		return fmt.Errorf("unknown resource %q", args[0])
	}
}

func forms(ctx context.Context, api *client.Client, args []string, jsonOutput bool, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("forms requires list, get, or update")
	}
	switch args[0] {
	case "list":
		f := flag.NewFlagSet("forms list", flag.ContinueOnError)
		f.SetOutput(io.Discard)
		search := f.String("search", "", "search form keys and names")
		formType := f.String("type", "", "filter by form type")
		page := f.Int("page", 0, "one-based page")
		perPage := f.Int("per-page", 20, "items per page")
		if err := f.Parse(args[1:]); err != nil {
			return err
		}
		result, err := api.ListForms(ctx, *search, *formType, *page, *perPage)
		if err != nil {
			return err
		}
		if jsonOutput {
			return writeJSON(out, result)
		}
		for _, item := range result.Data {
			status := "inactive"
			if item.Active {
				status = "active"
			}
			fmt.Fprintf(out, "%s\t%s\t%s\tv%d\t%s\t%d fields\t%s\n", item.Key, item.Name, item.Type, item.Version, status, item.FieldCount, item.Links.Admin)
		}
		return nil
	case "get":
		key, err := exactID(args, "get")
		if err != nil {
			return err
		}
		result, err := api.GetForm(ctx, key)
		if err != nil {
			return err
		}
		return printJSONOrForm(out, result.Data, jsonOutput)
	case "update":
		return updateForm(ctx, api, args, jsonOutput, out)
	default:
		return fmt.Errorf("unknown forms command %q", args[0])
	}
}

func updateForm(ctx context.Context, api *client.Client, args []string, jsonOutput bool, out io.Writer) error {
	key, rest, err := idAndRest(args)
	if err != nil {
		return err
	}
	f := flag.NewFlagSet("forms update", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	active := f.String("active", "", "set active to true or false")
	fieldsJSON := f.String("fields-json", "", "JSON array containing the complete form fields")
	if err := f.Parse(rest); err != nil {
		return err
	}
	payload := map[string]any{}
	if strings.TrimSpace(*active) != "" {
		value, parseErr := strconv.ParseBool(*active)
		if parseErr != nil {
			return errors.New("--active must be true or false")
		}
		payload["active"] = value
	}
	if strings.TrimSpace(*fieldsJSON) != "" {
		var fields []models.FormField
		if err := json.Unmarshal([]byte(*fieldsJSON), &fields); err != nil {
			return fmt.Errorf("--fields-json must be a valid JSON array: %w", err)
		}
		payload["fields"] = fields
	}
	if len(payload) == 0 {
		return errors.New("provide --active and/or --fields-json")
	}
	result, err := api.UpdateForm(ctx, key, payload)
	if err != nil {
		return err
	}
	return printJSONOrForm(out, result.Data, jsonOutput)
}

func faq(ctx context.Context, api *client.Client, args []string, jsonOutput bool, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("faq requires categories or faqs")
	}
	switch args[0] {
	case "categories":
		return categories(ctx, api, args[1:], jsonOutput, out)
	case "faqs":
		return faqs(ctx, api, args[1:], jsonOutput, out)
	default:
		return fmt.Errorf("unknown faq resource %q", args[0])
	}
}

func authentication(ctx context.Context, args []string, jsonOutput bool, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("auth requires login, status, or logout")
	}
	store := auth.NewStore()
	switch args[0] {
	case "login":
		f := flag.NewFlagSet("auth login", flag.ContinueOnError)
		f.SetOutput(io.Discard)
		email := f.String("email", "", "pre-fill the email on the local login page")
		apiURL := f.String("api-url", client.DefaultAPIBase, "Redan API URL override")
		if err := f.Parse(args[1:]); err != nil {
			return err
		}
		return runBrowserLogin(ctx, configuredAPIURL(*apiURL), *email, jsonOutput, out)
	case "status":
		credential, err := store.Load()
		if errors.Is(err, auth.ErrNotFound) {
			if jsonOutput {
				return writeJSON(out, map[string]any{"authenticated": false})
			}
			_, printErr := fmt.Fprintln(out, "Not logged in to Redan.")
			return printErr
		}
		if err != nil {
			return err
		}
		api, err := client.NewAuthenticated()
		if err != nil {
			return err
		}
		status, err := api.Status(ctx)
		if err != nil {
			return err
		}
		if jsonOutput {
			return writeJSON(out, status)
		}
		_, err = fmt.Fprintf(out, "Logged in as %s (%s). Session expires %s.\n", credential.UserName, credential.UserEmail, credential.ExpiresAt.Format(time.RFC3339))
		return err
	case "logout":
		if api, loadErr := client.NewAuthenticated(); loadErr == nil {
			_ = api.Logout(ctx)
		}
		if err := store.Delete(); err != nil {
			return fmt.Errorf("remove Redan session: %w", err)
		}
		if jsonOutput {
			return writeJSON(out, map[string]any{"logged_out": true})
		}
		_, err := fmt.Fprintln(out, "Redan session removed from the OS keyring.")
		return err
	default:
		return fmt.Errorf("unknown auth command %q", args[0])
	}
}

func configuredAPIURL(value string) string {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" {
		return client.DefaultAPIBase
	}
	return value
}

func categories(ctx context.Context, api *client.Client, args []string, jsonOutput bool, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("categories requires list, get, create, update, or delete")
	}
	switch args[0] {
	case "list":
		f := flag.NewFlagSet("categories list", flag.ContinueOnError)
		f.SetOutput(io.Discard)
		search := f.String("search", "", "search category names")
		page := f.Int("page", 0, "one-based page")
		perPage := f.Int("per-page", 20, "items per page")
		if err := f.Parse(args[1:]); err != nil {
			return err
		}
		result, err := api.ListCategories(ctx, *search, *page, *perPage)
		if err != nil {
			return err
		}
		if jsonOutput {
			return writeJSON(out, result)
		}
		for _, item := range result.Data {
			fmt.Fprintf(out, "%s\t%s\t%d FAQs\t%s\n", item.ID, item.Name, item.FAQCount, item.Links.Admin)
		}
		return nil
	case "get":
		id, err := exactID(args, "get")
		if err != nil {
			return err
		}
		result, err := api.GetCategory(ctx, id)
		if err != nil {
			return err
		}
		return printJSONOrCategory(out, result.Data, jsonOutput)
	case "create":
		f := flag.NewFlagSet("categories create", flag.ContinueOnError)
		f.SetOutput(io.Discard)
		id := f.String("id", "", "optional category ID")
		name := f.String("name", "", "category name")
		if err := f.Parse(args[1:]); err != nil {
			return err
		}
		if strings.TrimSpace(*name) == "" {
			return errors.New("--name is required")
		}
		result, err := api.CreateCategory(ctx, *id, *name)
		if err != nil {
			return err
		}
		return printJSONOrCategory(out, result.Data, jsonOutput)
	case "update":
		id, rest, err := idAndRest(args)
		if err != nil {
			return err
		}
		f := flag.NewFlagSet("categories update", flag.ContinueOnError)
		f.SetOutput(io.Discard)
		name := f.String("name", "", "category name")
		if err := f.Parse(rest); err != nil {
			return err
		}
		if strings.TrimSpace(*name) == "" {
			return errors.New("--name is required")
		}
		result, err := api.UpdateCategory(ctx, id, *name)
		if err != nil {
			return err
		}
		return printJSONOrCategory(out, result.Data, jsonOutput)
	case "delete":
		id, err := deleteID(args)
		if err != nil {
			return err
		}
		if !hasConfirm(args[1:]) {
			return errors.New("delete requires --confirm")
		}
		result, err := api.DeleteCategory(ctx, id)
		if err != nil {
			return err
		}
		return writeResult(out, result, jsonOutput, "Category deleted: "+id)
	default:
		return fmt.Errorf("unknown categories command %q", args[0])
	}
}

func faqs(ctx context.Context, api *client.Client, args []string, jsonOutput bool, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("faqs requires list, get, create, update, or delete")
	}
	switch args[0] {
	case "list":
		f := flag.NewFlagSet("faqs list", flag.ContinueOnError)
		f.SetOutput(io.Discard)
		search := f.String("search", "", "search questions and answers")
		category := f.String("category", "", "filter by category ID")
		page := f.Int("page", 0, "one-based page")
		perPage := f.Int("per-page", 20, "items per page")
		if err := f.Parse(args[1:]); err != nil {
			return err
		}
		result, err := api.ListFAQs(ctx, *search, *category, *page, *perPage)
		if err != nil {
			return err
		}
		if jsonOutput {
			return writeJSON(out, result)
		}
		for _, item := range result.Data {
			fmt.Fprintf(out, "%s\t%s\t%s\n%s\n", item.ID, item.CategoryName, item.Question["en"], item.Links.Admin)
		}
		return nil
	case "get":
		id, err := exactID(args, "get")
		if err != nil {
			return err
		}
		result, err := api.GetFAQ(ctx, id)
		if err != nil {
			return err
		}
		return printJSONOrFAQ(out, result.Data, jsonOutput)
	case "create", "update":
		return createOrUpdateFAQ(ctx, api, args, jsonOutput, out)
	case "delete":
		id, err := deleteID(args)
		if err != nil {
			return err
		}
		if !hasConfirm(args[1:]) {
			return errors.New("delete requires --confirm")
		}
		result, err := api.DeleteFAQ(ctx, id)
		if err != nil {
			return err
		}
		return writeResult(out, result, jsonOutput, "FAQ deleted: "+id)
	default:
		return fmt.Errorf("unknown faqs command %q", args[0])
	}
}

func createOrUpdateFAQ(ctx context.Context, api *client.Client, args []string, jsonOutput bool, out io.Writer) error {
	command := args[0]
	rest := args[1:]
	id := ""
	if command == "update" {
		var err error
		id, rest, err = idAndRest(args)
		if err != nil {
			return err
		}
	}
	f := flag.NewFlagSet("faqs "+command, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	faqID := f.String("id", "", "optional FAQ ID (create only)")
	category := f.String("category", "", "category ID")
	qe := f.String("question-en", "", "English question")
	qs := f.String("question-sn", "", "ChiShona question")
	qn := f.String("question-nd", "", "isiNdebele question")
	ae := f.String("answer-en", "", "English answer")
	as := f.String("answer-sn", "", "ChiShona answer")
	an := f.String("answer-nd", "", "isiNdebele answer")
	if err := f.Parse(rest); err != nil {
		return err
	}
	for name, value := range map[string]string{"--category": *category, "--question-en": *qe, "--question-sn": *qs, "--question-nd": *qn, "--answer-en": *ae, "--answer-sn": *as, "--answer-nd": *an} {
		if strings.TrimSpace(value) == "" {
			return errors.New(name + " is required")
		}
	}
	payload := map[string]any{"category": *category, "question": map[string]string{"en": *qe, "sn": *qs, "nd": *qn}, "answer": map[string]string{"en": *ae, "sn": *as, "nd": *an}}
	if command == "create" && *faqID != "" {
		payload["id"] = *faqID
	}
	var result models.FAQResponse
	var err error
	if command == "create" {
		result, err = api.CreateFAQ(ctx, payload)
	} else {
		result, err = api.UpdateFAQ(ctx, id, payload)
	}
	if err != nil {
		return err
	}
	return printJSONOrFAQ(out, result.Data, jsonOutput)
}

func takeJSON(args []string) (bool, []string) {
	result := make([]string, 0, len(args))
	jsonOutput := false
	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
		} else {
			result = append(result, arg)
		}
	}
	return jsonOutput, result
}
func exactID(args []string, command string) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("%s requires exactly one ID", command)
	}
	if args[1] == "" {
		return "", errors.New("ID cannot be empty")
	}
	return args[1], nil
}

func deleteID(args []string) (string, error) {
	if len(args) < 2 || args[1] == "" {
		return "", errors.New("delete requires exactly one ID and --confirm")
	}
	confirmed := false
	for _, arg := range args[2:] {
		if arg == "--confirm" {
			confirmed = true
			continue
		}
		if arg != "--confirm" {
			return "", errors.New("delete accepts only --confirm after the ID")
		}
	}
	if !confirmed {
		return "", errors.New("delete requires --confirm")
	}
	return args[1], nil
}
func idAndRest(args []string) (string, []string, error) {
	if len(args) < 2 || args[1] == "" {
		return "", nil, errors.New("command requires an ID")
	}
	return args[1], args[2:], nil
}
func hasConfirm(args []string) bool {
	for _, arg := range args {
		if arg == "--confirm" {
			return true
		}
	}
	return false
}
func printJSONOrCategory(out io.Writer, value models.Category, jsonOutput bool) error {
	if jsonOutput {
		return writeJSON(out, map[string]any{"data": value})
	}
	_, err := fmt.Fprintf(out, "%s\n%s\n%s\n", value.Name, value.Links.Admin, value.ID)
	return err
}
func printJSONOrFAQ(out io.Writer, value models.FAQ, jsonOutput bool) error {
	if jsonOutput {
		return writeJSON(out, map[string]any{"data": value})
	}
	_, err := fmt.Fprintf(out, "%s\n%s\n%s\n", value.Question["en"], value.Answer["en"], value.Links.Admin)
	return err
}
func printJSONOrForm(out io.Writer, value models.Form, jsonOutput bool) error {
	if jsonOutput {
		return writeJSON(out, map[string]any{"data": value})
	}
	_, err := fmt.Fprintf(out, "%s\n%s\nv%d (%d fields)\n%s\n", value.Name, value.Links.Admin, value.Version, value.FieldCount, value.Key)
	return err
}
func writeResult(out io.Writer, value any, jsonOutput bool, message string) error {
	if jsonOutput {
		return writeJSON(out, value)
	}
	_, err := fmt.Fprintln(out, message)
	return err
}
func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
func usage(out io.Writer) error {
	_, err := fmt.Fprintln(out, "redan manages Redan content, forms, and multilingual FAQs.\n\nCommands:\n  redan auth login [--email <address>]\n  redan auth status\n  redan auth logout\n  redan [--json] faq categories list|get|create|update|delete\n  redan [--json] faq faqs list|get|create|update|delete\n  redan [--json] forms list|get|update\n  redan version")
	return err
}
