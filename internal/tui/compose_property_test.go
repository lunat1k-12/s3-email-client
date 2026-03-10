package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"s3emailclient/internal/response"
)

// Feature: email-response, Property 8: Original Headers Display
// Validates: Requirements 3.4
func TestProperty_OriginalHeadersDisplay(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("compose view contains all four required header fields", prop.ForAll(
		func(from, subject string, to []string, date time.Time) bool {
			// Create a Model with compose data
			m := &Model{
				width:  80,
				height: 24,
				composeMode: true,
				composeData: &response.ComposeData{
					To:      "reply@example.com",
					Subject: "Re: " + subject,
					Body:    "",
					OriginalEmail: &response.OriginalEmailContext{
						From:    from,
						To:      to,
						Date:    date,
						Subject: subject,
					},
				},
				composeInput: initComposeTextarea(),
			}

			// Render the compose view
			rendered := m.renderComposeView()

			// Verify the rendered output contains all four required header fields
			hasFrom := strings.Contains(rendered, "From:") && strings.Contains(rendered, from)
			hasTo := strings.Contains(rendered, "To:")
			hasDate := strings.Contains(rendered, "Date:")
			hasSubject := strings.Contains(rendered, "Subject:") && strings.Contains(rendered, subject)

			// All four fields must be present
			return hasFrom && hasTo && hasDate && hasSubject
		},
		genEmailAddress(),
		genSubject(),
		genEmailAddressList(),
		genDate(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genEmailAddress generates random email addresses
func genEmailAddress() gopter.Gen {
	return gen.Identifier().Map(func(local string) string {
		if local == "" {
			local = "user"
		}
		return local + "@example.com"
	})
}

// genSubject generates random email subjects
func genSubject() gopter.Gen {
	return gen.AnyString().Map(func(s string) string {
		if s == "" {
			return "Test Subject"
		}
		// Limit length to avoid rendering issues
		if len(s) > 100 {
			return s[:100]
		}
		return s
	})
}

// genEmailAddressList generates a list of email addresses
func genEmailAddressList() gopter.Gen {
	return gen.SliceOfN(3, genEmailAddress()).Map(func(list []string) []string {
		if len(list) == 0 {
			return []string{"recipient@example.com"}
		}
		return list
	})
}

// genDate generates random dates within a reasonable range
func genDate() gopter.Gen {
	return gen.Int64Range(
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC).Unix(),
	).Map(func(timestamp int64) time.Time {
		return time.Unix(timestamp, 0).UTC()
	})
}
