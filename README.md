# Sensu Teams Handler
Make Sensu talk to MS Teams. Now with less ruby.

### Help output

Help:

```
The Sensu Go Teams handler

Usage:
  sensu-teams-handler [flags]
  sensu-teams-handler [command]

Available Commands:
  help        Help about any command
  version     Print the version number of this plugin

Flags:
  -h, --help                      help for sensu-teams-handler
  -i, --icon-url string           The URL for an icon to display in the message
  -l, --include-check-labels      Include check labels in Teams message?
  -e, --include-entity-labels     Include entity labels in Teams message?
  -t, --message-template string   The Teams notification output template, in Golang text/template format
  -r, --redact                    Enable redaction of labels
  -m, --redact-match string       Regex to redact values of matching labels (default "(?i).*(pass|key).*")
  -s, --summary-template string   The Teams summary template, in Golang text/template format (default "Sensu Event: {{.Entity.Name}}/{{.Check.Name}}: {{.Check.State}}")
  -w, --webhook-url string        The webhook url to send messages to

Use "sensu-teams-handler [command] --help" for more information about a command.
```

### Environment variables

|Argument                   |Environment Variable        |
|---------------------------|----------------------------|
|--webhook-url              |TEAMS_WEBHOOK_URL           |
|--icon-url                 |TEAMS_ICON_URL              |
|--message-template         |TEAMS_MESSAGE_TEMPLATE      |
|--summary-template         |TEAMS_SUMMARY_TEMPLATE      |
|--include-check-labels     |TEAMS_INCLUDE_CHECK_LABELS  |
|--include-entity-labels    |TEAMS_INCLUDE_ENTITY_LABELS |
|--redact-match             |TEAMS_REDACTMATCH           |
|--redact                   |TEAMS_REDACT                |

**Security Note:** Care should be taken to not expose the webhook URL for this handler by specifying it on the command line or by directly setting the environment variable in the handler definition.