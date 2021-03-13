package main

import (
	"bytes"
	"fmt"
	"regexp"

	goteamsnotify "github.com/atc0005/go-teams-notify/v2"
	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	"github.com/sensu-community/sensu-plugin-sdk/templates"
	corev2 "github.com/sensu/sensu-go/api/core/v2"
)

// HandlerConfig contains the Teams handler configuration
type HandlerConfig struct {
	sensu.PluginConfig
	teamsWebHookURL          string
	teamsIconURL             string
	teamsMessageTemplate     string
	teamsSummaryTemplate     string
	teamsRedactMatch         string
	teamsRedact              bool
	teamsIncludeCheckLabels  bool
	teamsIncludeEntityLabels bool
}

const (
	webHookURL      = "webhook-url"
	iconURL         = "icon-url"
	messageTemplate = "message-template"
	summaryTemplate = "summary-template"
	incCheckLabels  = "include-check-labels"
	incEntityLabels = "include-entity-labels"
	redactMatch     = "redact-match"
	redact          = "redact"
)

var (
	config = HandlerConfig{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-teams-handler",
			Short:    "The Sensu Go Teams handler",
			Keyspace: "sensu.io/plugins/teams/config",
		},
	}

	teamsConfigOptions = []*sensu.PluginConfigOption{
		{
			Path:      webHookURL,
			Env:       "TEAMS_WEBHOOK_URL",
			Argument:  webHookURL,
			Shorthand: "w",
			Secret:    true,
			Usage:     "The webhook url to send messages to",
			Value:     &config.teamsWebHookURL,
		},
		{
			Path:      iconURL,
			Env:       "TEAMS_ICON_URL",
			Argument:  iconURL,
			Shorthand: "i",
			Usage:     "The URL for an icon to display in the message",
			Value:     &config.teamsIconURL,
		},
		{
			Path:      messageTemplate,
			Env:       "TEAMS_MESSAGE_TEMPLATE",
			Argument:  messageTemplate,
			Shorthand: "t",
			Usage:     "The Teams notification output template, in Golang text/template format",
			Value:     &config.teamsMessageTemplate,
		},
		{
			Path:      summaryTemplate,
			Env:       "TEAMS_SUMMARY_TEMPLATE",
			Argument:  summaryTemplate,
			Shorthand: "s",
			Default:   "Sensu Event: {{.Entity.Name}}/{{.Check.Name}}: {{.Check.State}}",
			Usage:     "The Teams summary template, in Golang text/template format",
			Value:     &config.teamsSummaryTemplate,
		},
		{
			Path:      redactMatch,
			Env:       "TEAMS_REDACTMATCH",
			Argument:  redactMatch,
			Shorthand: "m",
			Default:   "(?i).*(pass|key).*",
			Usage:     "Regex to redact values of matching labels",
			Value:     &config.teamsRedactMatch,
		},
		{
			Path:      redact,
			Env:       "TEAMS_REDACT",
			Argument:  redact,
			Shorthand: "r",
			Default:   false,
			Usage:     "Enable redaction of labels",
			Value:     &config.teamsRedact,
		},
		{
			Path:      incCheckLabels,
			Env:       "TEAMS_INCLUDE_CHECK_LABELS",
			Argument:  incCheckLabels,
			Shorthand: "l",
			Default:   false,
			Usage:     "Include check labels in Teams message?",
			Value:     &config.teamsIncludeCheckLabels,
		},
		{
			Path:      incEntityLabels,
			Env:       "TEAMS_INCLUDE_ENTITY_LABELS",
			Argument:  incEntityLabels,
			Shorthand: "e",
			Default:   false,
			Usage:     "Include entity labels in Teams message?",
			Value:     &config.teamsIncludeEntityLabels,
		},
	}
)

func main() {
	goHandler := sensu.NewGoHandler(&config.PluginConfig, teamsConfigOptions, checkArgs, sendMessage)
	goHandler.Execute()
}

func checkArgs(_ *corev2.Event) (e error) {
	if len(config.teamsWebHookURL) == 0 {
		return fmt.Errorf("--%s or TEAMS_WEBHOOK_URL environment variable is required", webHookURL)
	}

	// validate the regex compiles, if not catch the panic and return error
	defer func() {
		if r := recover(); r != nil {
			e = fmt.Errorf("regexp (%s) specified by TEAMS_REDACT or --redact is invalid", config.teamsRedactMatch)
		}
		return
	}()
	regexp.MustCompile(config.teamsRedactMatch)

	return nil
}

func messageColor(event *corev2.Event) string {
	switch event.Check.Status {
	case 0:
		// Green
		return "#008450"
	case 2:
		// Red
		return "#B81D13"
	default:
		// Yellow
		return "#EFB700"
	}
}

func messageStatus(event *corev2.Event) string {
	switch event.Check.Status {
	case 0:
		return "Resolved"
	case 2:
		return "Critical"
	default:
		return "Warning"
	}
}

func createMessage(event *corev2.Event) goteamsnotify.MessageCard {
	var message string

	summary, err := templates.EvalTemplate("summary", config.teamsSummaryTemplate, event)
	if err != nil {
		fmt.Printf("%s: Error processing summary template: %s", config.PluginConfig.Name, err)
	}
	if config.teamsMessageTemplate != "" {
		message, err = templates.EvalTemplate("message", config.teamsMessageTemplate, event)
		if err != nil {
			fmt.Printf("%s: Error processing message template: %s", config.PluginConfig.Name, err)
		}
	}

	msgCard := goteamsnotify.NewMessageCard()
	msgCard.Title = fmt.Sprintf("Sensu Event (%s)", messageStatus(event))
	msgCard.Summary = summary
	msgCard.ThemeColor = messageColor(event)
	msgCard.Text = message
	msgCardSection := goteamsnotify.MessageCardSection{
		ActivityImage: config.teamsIconURL,
	}
	if message == "" {
		msgCardSection.AddFact(goteamsnotify.MessageCardSectionFact{
			Name:  "Entity",
			Value: event.Entity.Name,
		})
		msgCardSection.AddFact(goteamsnotify.MessageCardSectionFact{
			Name:  "Check",
			Value: event.Check.Name,
		})
		msgCardSection.AddFact(goteamsnotify.MessageCardSectionFact{
			Name:  "State",
			Value: event.Check.State,
		})
		msgCardSection.AddFact(goteamsnotify.MessageCardSectionFact{
			Name:  "Occurrences",
			Value: fmt.Sprintf("%d", event.Check.Occurrences),
		})
		msgCardSection.AddFact(goteamsnotify.MessageCardSectionFact{
			Name:  "Output",
			Value: fmt.Sprintf("```\n%s```", event.Check.Output),
		})
	}

	if config.teamsIncludeEntityLabels && event.Entity.Labels != nil {
		re := regexp.MustCompile(config.teamsRedactMatch)
		buf := bytes.Buffer{}
		for k, v := range event.Entity.Labels {
			if config.teamsRedact && re.MatchString(k) {
				v = "**REDACTED**"
			}
			fmt.Fprintf(&buf, "%s: %s\n", k, v)
		}
		msgCardSection.AddFact(goteamsnotify.MessageCardSectionFact{
			Name:  "Entity Labels",
			Value: buf.String(),
		})
	}

	if config.teamsIncludeCheckLabels && event.Check.Labels != nil {
		re := regexp.MustCompile(config.teamsRedactMatch)
		buf := bytes.Buffer{}
		for k, v := range event.Check.Labels {
			if config.teamsRedact && re.MatchString(k) {
				v = "**REDACTED**"
			}
			fmt.Fprintf(&buf, "%s: %s\n<br>", k, v)
		}
		msgCardSection.AddFact(goteamsnotify.MessageCardSectionFact{
			Name:  "Check Labels",
			Value: buf.String(),
		})
	}
	msgCard.AddSection(&msgCardSection)

	return msgCard
}

func sendMessage(event *corev2.Event) error {
	mstClient := goteamsnotify.NewClient()
	mstClient.SkipWebhookURLValidationOnSend(true)
	msgCard := createMessage(event)
	err := mstClient.Send(config.teamsWebHookURL, msgCard)
	if err != nil {
		return fmt.Errorf("Failed to send Teams message: %v", err)
	}

	fmt.Print("Notification sent to Teams.\n")

	return nil
}
