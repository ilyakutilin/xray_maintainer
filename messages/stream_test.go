package messages

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

// captureOutput captures stdout during function execution
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestStreamSender_Send(t *testing.T) {
	tests := []struct {
		name     string
		message  Message
		expected string
	}{
		{
			name: "no errors",
			message: Message{
				Subject: "Test Subject",
				Body:    "Test Body",
				Errors:  []error{},
			},
			expected: `The following message would be sent:
Subject: Test Subject
Body: Test Body
There are no errors.
`,
		},
		{
			name: "single error",
			message: Message{
				Subject: "Single Error",
				Body:    "Error Body",
				Errors:  []error{fmt.Errorf("test error")},
			},
			expected: `The following message would be sent:
Subject: Single Error
Body: Error Body
Error: test error
`,
		},
		{
			name: "multiple errors",
			message: Message{
				Subject: "Multi Error",
				Body:    "Multi Body",
				Errors: []error{
					fmt.Errorf("first error"),
					fmt.Errorf("second error"),
				},
			},
			expected: `The following message would be sent:
Subject: Multi Error
Body: Multi Body
There are 2 errors:
0) first error
1) second error
`,
		},
		{
			name: "empty message",
			message: Message{
				Subject: "",
				Body:    "",
				Errors:  []error{},
			},
			expected: `The following message would be sent:
Subject: 
Body: 
There are no errors.
`,
		},
	}

	sender := &StreamSender{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				if err := sender.Send(tt.message); err != nil {
					t.Errorf("Send() returned unexpected error: %v", err)
				}
			})

			if output != tt.expected {
				t.Errorf("unexpected output:\ngot:\n%v\nwant:\n%v", output, tt.expected)
			}
		})
	}
}
