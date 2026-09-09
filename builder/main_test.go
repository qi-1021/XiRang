package main

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestReadLine(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		prompt         string
		defaultValue   string
		expectedResult string
		expectedPrompt string
	}{
		{
			name:           "User provides custom input",
			input:          "custom_val\n",
			prompt:         "Enter value",
			defaultValue:   "default_val",
			expectedResult: "custom_val",
			expectedPrompt: "Enter value [default_val]: ",
		},
		{
			name:           "User provides empty input with default value",
			input:          "\n",
			prompt:         "Enter value",
			defaultValue:   "default_val",
			expectedResult: "default_val",
			expectedPrompt: "Enter value [default_val]: ",
		},
		{
			name:           "User provides empty input without default value",
			input:          "\n",
			prompt:         "Enter value",
			defaultValue:   "",
			expectedResult: "",
			expectedPrompt: "Enter value: ",
		},
		{
			name:           "User provides input with leading and trailing spaces",
			input:          "   trimmed_value   \n",
			prompt:         "Enter value",
			defaultValue:   "default_val",
			expectedResult: "trimmed_value",
			expectedPrompt: "Enter value [default_val]: ",
		},
		{
			name:           "EOF or no newline in input",
			input:          "no_newline",
			prompt:         "Enter value",
			defaultValue:   "default_val",
			expectedResult: "no_newline",
			expectedPrompt: "Enter value [default_val]: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout to verify prompt output
			oldStdout := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("failed to create pipe: %v", err)
			}
			os.Stdout = w

			reader := bufio.NewReader(strings.NewReader(tt.input))
			result := readLine(reader, tt.prompt, tt.defaultValue)

			_ = w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			_ = r.Close()
			actualPrompt := buf.String()

			if result != tt.expectedResult {
				t.Errorf("readLine() result = %q, expected %q", result, tt.expectedResult)
			}

			if actualPrompt != tt.expectedPrompt {
				t.Errorf("readLine() prompt output = %q, expected %q", actualPrompt, tt.expectedPrompt)
			}
		})
	}
}
