package slack

import (
	"github.com/slack-go/slack"
	"github.com/ssm-access-manager/internal/models"
	"github.com/ssm-access-manager/internal/repository"
)

// BuildAccessRequestModal creates a modal view for SSM access requests
func BuildAccessRequestModal(accounts []repository.AccountOption, managerGroups []*models.ApprovalGroup) slack.ModalViewRequest {
	// Build account options for dropdown
	var accountOptions []*slack.OptionBlockObject
	for _, account := range accounts {
		accountOptions = append(accountOptions, &slack.OptionBlockObject{
			Text: &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: account.Text,
			},
			Value: account.Value,
		})
	}

	// Build manager group options for dropdown
	var managerGroupOptions []*slack.OptionBlockObject
	for _, group := range managerGroups {
		managerGroupOptions = append(managerGroupOptions, &slack.OptionBlockObject{
			Text: &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: group.GroupName,
			},
			Value: group.GroupID,
		})
	}

	// Build modal blocks
	blocks := slack.Blocks{
		BlockSet: []slack.Block{
			// Account selection dropdown
			slack.NewInputBlock(
				"account_block",
				&slack.TextBlockObject{
					Type: slack.PlainTextType,
					Text: "Account",
				},
				nil,
				&slack.SelectBlockElement{
					Type:        slack.OptTypeStatic,
					ActionID:    "account_select",
					Placeholder: &slack.TextBlockObject{Type: slack.PlainTextType, Text: "Select an account"},
					Options:     accountOptions,
				},
			),
			// Manager group selection dropdown
			slack.NewInputBlock(
				"manager_group_block",
				&slack.TextBlockObject{
					Type: slack.PlainTextType,
					Text: "Manager Group",
				},
				&slack.TextBlockObject{
					Type: slack.PlainTextType,
					Text: "Select the manager group that should approve this request",
				},
				&slack.SelectBlockElement{
					Type:        slack.OptTypeStatic,
					ActionID:    "manager_group_select",
					Placeholder: &slack.TextBlockObject{Type: slack.PlainTextType, Text: "Select a manager group"},
					Options:     managerGroupOptions,
				},
			),
			// Host input
			slack.NewInputBlock(
				"host_block",
				&slack.TextBlockObject{
					Type: slack.PlainTextType,
					Text: "Host",
				},
				nil,
				&slack.PlainTextInputBlockElement{
					Type:     slack.METPlainTextInput,
					ActionID: "host_input",
					Placeholder: &slack.TextBlockObject{
						Type: slack.PlainTextType,
						Text: "db.example.com or 10.0.1.100",
					},
				},
			),
			// Port input
			slack.NewInputBlock(
				"port_block",
				&slack.TextBlockObject{
					Type: slack.PlainTextType,
					Text: "Port",
				},
				nil,
				&slack.PlainTextInputBlockElement{
					Type:         slack.METPlainTextInput,
					ActionID:     "port_input",
					InitialValue: "22",
					Placeholder: &slack.TextBlockObject{
						Type: slack.PlainTextType,
						Text: "22",
					},
				},
			),
			// Reason input
			slack.NewInputBlock(
				"reason_block",
				&slack.TextBlockObject{
					Type: slack.PlainTextType,
					Text: "Reason for Access",
				},
				nil,
				&slack.PlainTextInputBlockElement{
					Type:      slack.METPlainTextInput,
					ActionID:  "reason_input",
					Multiline: true,
					Placeholder: &slack.TextBlockObject{
						Type: slack.PlainTextType,
						Text: "Explain why you need this access (required)",
					},
				},
			),
			// Expiration date input (optional)
			slack.NewInputBlock(
				"expires_block",
				&slack.TextBlockObject{
					Type: slack.PlainTextType,
					Text: "Expiration Date (optional)",
				},
				&slack.TextBlockObject{
					Type: slack.PlainTextType,
					Text: "Leave empty to default to 14 days from now",
				},
				&slack.PlainTextInputBlockElement{
					Type:     slack.METPlainTextInput,
					ActionID: "expires_input",
					Placeholder: &slack.TextBlockObject{
						Type: slack.PlainTextType,
						Text: "YYYY-MM-DD (e.g., 2026-03-15)",
					},
				},
			).WithOptional(true),
		},
	}

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		CallbackID: "ssm_access_request",
		Title: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Request SSM Access",
		},
		Submit: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Submit",
		},
		Close: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Cancel",
		},
		Blocks: blocks,
	}
}
