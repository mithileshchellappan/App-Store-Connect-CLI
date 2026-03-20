package web

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
	webcore "github.com/rudrankriyam/App-Store-Connect-CLI/internal/web"
)

type reviewSubscriptionsListOutput struct {
	AppID         string                       `json:"appId"`
	AttachedCount int                          `json:"attachedCount"`
	Subscriptions []webcore.ReviewSubscription `json:"subscriptions"`
}

type reviewSubscriptionMutationOutput struct {
	AppID        string                     `json:"appId"`
	Operation    string                     `json:"operation"`
	Changed      bool                       `json:"changed"`
	SubmissionID string                     `json:"submissionId,omitempty"`
	Subscription webcore.ReviewSubscription `json:"subscription"`
}

func reviewSubscriptionValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "n/a"
	}
	return trimmed
}

func reviewSubscriptionName(subscription webcore.ReviewSubscription) string {
	switch {
	case strings.TrimSpace(subscription.Name) != "":
		return strings.TrimSpace(subscription.Name)
	case strings.TrimSpace(subscription.ProductID) != "":
		return strings.TrimSpace(subscription.ProductID)
	default:
		return strings.TrimSpace(subscription.ID)
	}
}

func reviewSubscriptionBool(value bool) string {
	return strconv.FormatBool(value)
}

func reviewSubscriptionAttachPreflight(appID string, subscription webcore.ReviewSubscription) error {
	state := strings.ToUpper(strings.TrimSpace(subscription.State))
	if state != "MISSING_METADATA" {
		return nil
	}

	subscriptionID := strings.TrimSpace(subscription.ID)
	if subscriptionID == "" {
		subscriptionID = "SUB_ID"
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintf(
		os.Stderr,
		"Attach preflight: subscription %q (%s) is %s, so Apple will not attach it to the next app version review yet.\n",
		reviewSubscriptionName(subscription),
		reviewSubscriptionValue(subscriptionID),
		state,
	)
	fmt.Fprintf(
		os.Stderr,
		"Hint: run `asc validate subscriptions --app \"%s\"` to inspect readiness.\n",
		reviewSubscriptionValue(strings.TrimSpace(appID)),
	)
	fmt.Fprintln(os.Stderr, "Hint: Apple only allows this attach flow after the subscription reaches READY_TO_SUBMIT.")
	fmt.Fprintln(os.Stderr, "Hint: Check localizations, pricing coverage, and the App Store review screenshot.")
	fmt.Fprintf(
		os.Stderr,
		"Hint: In live testing, a subscription promotional image also mattered even though App Store Connect surfaces it as a recommendation. Upload one with `asc subscriptions images create --subscription-id \"%s\" --file \"./image.png\"` if it is missing.\n",
		reviewSubscriptionValue(subscriptionID),
	)

	return shared.NewReportedError(
		fmt.Errorf(
			"web review subscriptions attach: subscription %q is %s; Apple only allows attach once it reaches READY_TO_SUBMIT",
			subscriptionID,
			state,
		),
	)
}

func countAttachedReviewSubscriptions(subscriptions []webcore.ReviewSubscription) int {
	count := 0
	for _, subscription := range subscriptions {
		if subscription.SubmitWithNextAppStoreVersion {
			count++
		}
	}
	return count
}

func buildReviewSubscriptionsListTableRows(subscriptions []webcore.ReviewSubscription) [][]string {
	rows := make([][]string, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		rows = append(rows, []string{
			reviewSubscriptionValue(subscription.GroupReferenceName),
			reviewSubscriptionValue(subscription.ID),
			reviewSubscriptionValue(subscription.ProductID),
			reviewSubscriptionValue(reviewSubscriptionName(subscription)),
			reviewSubscriptionValue(subscription.State),
			reviewSubscriptionBool(subscription.SubmitWithNextAppStoreVersion),
			reviewSubscriptionBool(subscription.IsAppStoreReviewInProgress),
		})
	}
	return rows
}

func renderReviewSubscriptionsListTable(payload reviewSubscriptionsListOutput) error {
	headers := []string{"Group", "Subscription ID", "Product ID", "Name", "State", "Next Version", "Review In Progress"}
	asc.RenderTable(headers, buildReviewSubscriptionsListTableRows(payload.Subscriptions))
	return nil
}

func renderReviewSubscriptionsListMarkdown(payload reviewSubscriptionsListOutput) error {
	headers := []string{"Group", "Subscription ID", "Product ID", "Name", "State", "Next Version", "Review In Progress"}
	asc.RenderMarkdown(headers, buildReviewSubscriptionsListTableRows(payload.Subscriptions))
	return nil
}

func buildReviewSubscriptionMutationRows(payload reviewSubscriptionMutationOutput) [][]string {
	return [][]string{
		{"Mutation", "App ID", reviewSubscriptionValue(payload.AppID)},
		{"Mutation", "Operation", reviewSubscriptionValue(payload.Operation)},
		{"Mutation", "Changed", reviewSubscriptionBool(payload.Changed)},
		{"Mutation", "Submission ID", reviewSubscriptionValue(payload.SubmissionID)},
		{"Subscription", "Subscription ID", reviewSubscriptionValue(payload.Subscription.ID)},
		{"Subscription", "Product ID", reviewSubscriptionValue(payload.Subscription.ProductID)},
		{"Subscription", "Name", reviewSubscriptionValue(reviewSubscriptionName(payload.Subscription))},
		{"Subscription", "Group", reviewSubscriptionValue(payload.Subscription.GroupReferenceName)},
		{"Subscription", "State", reviewSubscriptionValue(payload.Subscription.State)},
		{"Subscription", "Next Version", reviewSubscriptionBool(payload.Subscription.SubmitWithNextAppStoreVersion)},
		{"Subscription", "Review In Progress", reviewSubscriptionBool(payload.Subscription.IsAppStoreReviewInProgress)},
	}
}

func renderReviewSubscriptionMutationTable(payload reviewSubscriptionMutationOutput) error {
	headers := []string{"Section", "Field", "Value"}
	asc.RenderTable(headers, buildReviewSubscriptionMutationRows(payload))
	return nil
}

func renderReviewSubscriptionMutationMarkdown(payload reviewSubscriptionMutationOutput) error {
	headers := []string{"Section", "Field", "Value"}
	asc.RenderMarkdown(headers, buildReviewSubscriptionMutationRows(payload))
	return nil
}

func findReviewSubscription(subscriptions []webcore.ReviewSubscription, subscriptionID string) (*webcore.ReviewSubscription, bool) {
	subscriptionID = strings.TrimSpace(subscriptionID)
	for i := range subscriptions {
		if strings.TrimSpace(subscriptions[i].ID) == subscriptionID {
			match := subscriptions[i]
			return &match, true
		}
	}
	return nil, false
}

func loadReviewSubscriptionsWithLabel(ctx context.Context, client *webcore.Client, appID, label string) ([]webcore.ReviewSubscription, error) {
	return withWebSpinnerValue(label, func() ([]webcore.ReviewSubscription, error) {
		return client.ListReviewSubscriptions(ctx, appID)
	})
}

// WebReviewSubscriptionsCommand returns the app-version subscription attach helpers.
func WebReviewSubscriptionsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web review subscriptions", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "subscriptions",
		ShortUsage: "asc web review subscriptions <subcommand> [flags]",
		ShortHelp:  "[experimental] Inspect and mutate review-attached subscriptions.",
		LongHelp: `EXPERIMENTAL / UNOFFICIAL / DISCOURAGED

Inspect and mutate subscription selection for the next app version review.
This uses private Apple web-session /iris endpoints and may break without notice.

Subcommands:
  list    List subscriptions and their next-version attach state
  attach  Attach one subscription to the next app version review
  remove  Remove one subscription from the next app version review

` + webWarningText,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			WebReviewSubscriptionsListCommand(),
			WebReviewSubscriptionsAttachCommand(),
			WebReviewSubscriptionsRemoveCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// WebReviewSubscriptionsListCommand lists review-scoped subscriptions for an app.
func WebReviewSubscriptionsListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web review subscriptions list", flag.ExitOnError)

	appID := fs.String("app", "", "App ID")
	authFlags := bindWebSessionFlags(fs)
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "asc web review subscriptions list --app APP_ID [flags]",
		ShortHelp:  "[experimental] List subscriptions and next-version attach state.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			trimmedAppID := strings.TrimSpace(*appID)
			if trimmedAppID == "" {
				return shared.UsageError("--app is required")
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			session, err := resolveWebSessionForCommand(requestCtx, authFlags)
			if err != nil {
				return err
			}
			client := newWebClientFn(session)

			subscriptions, err := loadReviewSubscriptionsWithLabel(requestCtx, client, trimmedAppID, "Loading review subscriptions")
			if err != nil {
				return withWebAuthHint(err, "web review subscriptions list")
			}

			payload := reviewSubscriptionsListOutput{
				AppID:         trimmedAppID,
				AttachedCount: countAttachedReviewSubscriptions(subscriptions),
				Subscriptions: subscriptions,
			}
			return shared.PrintOutputWithRenderers(
				payload,
				*output.Output,
				*output.Pretty,
				func() error { return renderReviewSubscriptionsListTable(payload) },
				func() error { return renderReviewSubscriptionsListMarkdown(payload) },
			)
		},
	}
}

// WebReviewSubscriptionsAttachCommand attaches a subscription to the next app version review.
func WebReviewSubscriptionsAttachCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web review subscriptions attach", flag.ExitOnError)

	appID := fs.String("app", "", "App ID")
	subscriptionID := fs.String("subscription-id", "", "Subscription ID")
	confirm := fs.Bool("confirm", false, "Confirm the attach operation")
	authFlags := bindWebSessionFlags(fs)
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "attach",
		ShortUsage: "asc web review subscriptions attach --app APP_ID --subscription-id SUB_ID --confirm [flags]",
		ShortHelp:  "[experimental] Attach a subscription to the next app version review.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			trimmedAppID := strings.TrimSpace(*appID)
			trimmedSubscriptionID := strings.TrimSpace(*subscriptionID)
			switch {
			case trimmedAppID == "":
				return shared.UsageError("--app is required")
			case trimmedSubscriptionID == "":
				return shared.UsageError("--subscription-id is required")
			case !*confirm:
				return shared.UsageError("--confirm is required")
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			session, err := resolveWebSessionForCommand(requestCtx, authFlags)
			if err != nil {
				return err
			}
			client := newWebClientFn(session)

			subscriptions, err := loadReviewSubscriptionsWithLabel(requestCtx, client, trimmedAppID, "Loading review subscriptions")
			if err != nil {
				return withWebAuthHint(err, "web review subscriptions attach")
			}
			selected, ok := findReviewSubscription(subscriptions, trimmedSubscriptionID)
			if !ok {
				return fmt.Errorf("subscription %q was not found for app %q", trimmedSubscriptionID, trimmedAppID)
			}

			payload := reviewSubscriptionMutationOutput{
				AppID:        trimmedAppID,
				Operation:    "attach",
				Changed:      false,
				Subscription: *selected,
			}
			if !selected.SubmitWithNextAppStoreVersion {
				if err := reviewSubscriptionAttachPreflight(trimmedAppID, *selected); err != nil {
					return err
				}

				submission, err := withWebSpinnerValue("Attaching subscription to next app version", func() (webcore.ReviewSubscriptionSubmission, error) {
					return client.CreateSubscriptionSubmission(requestCtx, trimmedSubscriptionID)
				})
				if err != nil {
					return withWebAuthHint(err, "web review subscriptions attach")
				}

				refreshedSubscriptions, err := loadReviewSubscriptionsWithLabel(requestCtx, client, trimmedAppID, "Refreshing review subscriptions")
				if err != nil {
					return withWebAuthHint(err, "web review subscriptions attach")
				}
				refreshed, ok := findReviewSubscription(refreshedSubscriptions, trimmedSubscriptionID)
				if !ok {
					return fmt.Errorf("subscription %q was not found for app %q after attach", trimmedSubscriptionID, trimmedAppID)
				}
				payload.Changed = true
				payload.SubmissionID = strings.TrimSpace(submission.ID)
				payload.Subscription = *refreshed
			}

			return shared.PrintOutputWithRenderers(
				payload,
				*output.Output,
				*output.Pretty,
				func() error { return renderReviewSubscriptionMutationTable(payload) },
				func() error { return renderReviewSubscriptionMutationMarkdown(payload) },
			)
		},
	}
}

// WebReviewSubscriptionsRemoveCommand removes a subscription from the next app version review.
func WebReviewSubscriptionsRemoveCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web review subscriptions remove", flag.ExitOnError)

	appID := fs.String("app", "", "App ID")
	subscriptionID := fs.String("subscription-id", "", "Subscription ID")
	confirm := fs.Bool("confirm", false, "Confirm the remove operation")
	authFlags := bindWebSessionFlags(fs)
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "remove",
		ShortUsage: "asc web review subscriptions remove --app APP_ID --subscription-id SUB_ID --confirm [flags]",
		ShortHelp:  "[experimental] Remove a subscription from the next app version review.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			trimmedAppID := strings.TrimSpace(*appID)
			trimmedSubscriptionID := strings.TrimSpace(*subscriptionID)
			switch {
			case trimmedAppID == "":
				return shared.UsageError("--app is required")
			case trimmedSubscriptionID == "":
				return shared.UsageError("--subscription-id is required")
			case !*confirm:
				return shared.UsageError("--confirm is required")
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			session, err := resolveWebSessionForCommand(requestCtx, authFlags)
			if err != nil {
				return err
			}
			client := newWebClientFn(session)

			subscriptions, err := loadReviewSubscriptionsWithLabel(requestCtx, client, trimmedAppID, "Loading review subscriptions")
			if err != nil {
				return withWebAuthHint(err, "web review subscriptions remove")
			}
			selected, ok := findReviewSubscription(subscriptions, trimmedSubscriptionID)
			if !ok {
				return fmt.Errorf("subscription %q was not found for app %q", trimmedSubscriptionID, trimmedAppID)
			}

			payload := reviewSubscriptionMutationOutput{
				AppID:        trimmedAppID,
				Operation:    "remove",
				Changed:      false,
				Subscription: *selected,
			}
			if selected.SubmitWithNextAppStoreVersion {
				err = withWebSpinner("Removing subscription from next app version", func() error {
					return client.DeleteSubscriptionSubmission(requestCtx, trimmedSubscriptionID)
				})
				if err != nil {
					return withWebAuthHint(err, "web review subscriptions remove")
				}

				refreshedSubscriptions, err := loadReviewSubscriptionsWithLabel(requestCtx, client, trimmedAppID, "Refreshing review subscriptions")
				if err != nil {
					return withWebAuthHint(err, "web review subscriptions remove")
				}
				refreshed, ok := findReviewSubscription(refreshedSubscriptions, trimmedSubscriptionID)
				if !ok {
					return fmt.Errorf("subscription %q was not found for app %q after remove", trimmedSubscriptionID, trimmedAppID)
				}
				payload.Changed = true
				payload.Subscription = *refreshed
			}

			return shared.PrintOutputWithRenderers(
				payload,
				*output.Output,
				*output.Pretty,
				func() error { return renderReviewSubscriptionMutationTable(payload) },
				func() error { return renderReviewSubscriptionMutationMarkdown(payload) },
			)
		},
	}
}
