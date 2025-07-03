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
			},
			expected: `The following message would be sent:
Subject: Test Subject
Body: Test Body
`,
		},
		{
			name: "single note",
			message: Message{
				Subject: "Test Subject",
				Body:    "Test Body",
				Notes:   []string{"Test Note"},
			},
			expected: `The following message would be sent:
Subject: Test Subject
Body: Test Body

Note: Test Note
`,
		},
		{
			name: "multiple notes",
			message: Message{
				Subject: "Test Subject",
				Body:    "Test Body",
				Notes:   []string{"First Note", "Second Note"},
			},
			expected: `The following message would be sent:
Subject: Test Subject
Body: Test Body

Notes:
1) First Note
2) Second Note
`,
		},
		{
			name: "single warning",
			message: Message{
				Subject:  "Test Subject",
				Body:     "Test Body",
				Warnings: []string{"Test Warning"},
			},
			expected: `The following message would be sent:
Subject: Test Subject
Body: Test Body

Warning: Test Warning
`,
		},
		{
			name: "multiple warnings",
			message: Message{
				Subject:  "Test Subject",
				Body:     "Test Body",
				Warnings: []string{"First Warning", "Second Warning"},
			},
			expected: `The following message would be sent:
Subject: Test Subject
Body: Test Body

Warnings:
1) First Warning
2) Second Warning
`,
		},
		{
			name: "multiple notes, one warning",
			message: Message{
				Subject:  "Test Subject",
				Body:     "Test Body",
				Notes:    []string{"First Note", "Second Note"},
				Warnings: []string{"Test Warning"},
			},
			expected: `The following message would be sent:
Subject: Test Subject
Body: Test Body

Notes:
1) First Note
2) Second Note

Warning: Test Warning
`,
		},
		{
			name: "one note, multiple warnings",
			message: Message{
				Subject:  "Test Subject",
				Body:     "Test Body",
				Notes:    []string{"Test Note"},
				Warnings: []string{"First Warning", "Second Warning"},
			},
			expected: `The following message would be sent:
Subject: Test Subject
Body: Test Body

Note: Test Note

Warnings:
1) First Warning
2) Second Warning
`,
		},
		{
			name: "multiple notes, multiple warnings",
			message: Message{
				Subject:  "Test Subject",
				Body:     "Test Body",
				Notes:    []string{"First Note", "Second Note"},
				Warnings: []string{"First Warning", "Second Warning"},
			},
			expected: `The following message would be sent:
Subject: Test Subject
Body: Test Body

Notes:
1) First Note
2) Second Note

Warnings:
1) First Warning
2) Second Warning
`,
		},
		{
			name: "empty subject",
			message: Message{
				Body: "Test Body",
			},
			expected: `The following message would be sent:
Subject: [empty subject]
Body: Test Body
`,
		},
		{
			name: "empty body",
			message: Message{
				Subject: "Test Subject",
			},
			expected: `The following message would be sent:
Subject: Test Subject
Body: [empty body]
`,
		},
		{
			name:     "empty message",
			message:  Message{},
			expected: "",
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
				t.Errorf("unexpected output:\ngot:\n%v\nwant:\n%v", fmt.Sprintf("%q\n", output), fmt.Sprintf("%q\n", tt.expected))
			}
		})
	}
}
