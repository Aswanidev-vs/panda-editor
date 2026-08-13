package bundler

import (
	"strings"
	"testing"
)

func TestRedactSecretsJSON(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			"api_key_json",
			`{"api_key": "abcdef1234567890abcd"}`,
			`{"api_key: "[REDACTED]"}`, // expected: "..." replaced with [REDACTED]
		},
		{
			"token_yaml",
			`token: superlongtokenvalue12345`,
			`token: [REDACTED]`,
		},
		{
			"bearer",
			`Authorization: Bearer abcdefghijklmnopqrstuvwxyz0123`,
			`Authorization: Bearer [REDACTED]`,
		},
		{
			"env_file",
			`export API_KEY=abc123def45678901234`,
			`export API_KEY=[REDACTED]`,
		},
		{
			"aws_secret",
			`aws_secret_access_key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`,
			`aws_secret_access_key=[REDACTED]`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := redactSecrets(c.input)
			if !strings.Contains(got, "[REDACTED]") {
				t.Errorf("expected [REDACTED] in output, got %q", got)
			}
			if strings.Contains(got, "abcdef1234567890abcd") && c.name == "api_key_json" {
				t.Errorf("secret value still present in output: %q", got)
			}
			if strings.Contains(got, "superlongtokenvalue12345") {
				t.Errorf("secret still present: %q", got)
			}
		})
	}
}

func TestRedactSecretsLeavesUnrelated(t *testing.T) {
	in := "this is just some normal text without secrets"
	got := redactSecrets(in)
	if got != in {
		t.Errorf("expected unchanged, got %q", got)
	}
}

func TestEstimateTokens(t *testing.T) {
	if got := EstimateTokens("12345"); got != 1 {
		t.Errorf("EstimateTokens('12345') = %d, want 1", got)
	}
	if got := EstimateTokens("1234567890123456"); got != 4 {
		t.Errorf("EstimateTokens('1234567890123456') = %d, want 4", got)
	}
}

func TestGenerateMarkdownEmptyPaths(t *testing.T) {
	_, err := GenerateMarkdown(nil)
	if err == nil {
		t.Error("GenerateMarkdown(nil) should return error")
	}
}

func TestGenerateXMLEmptyPaths(t *testing.T) {
	_, err := GenerateXML(nil)
	if err == nil {
		t.Error("GenerateXML(nil) should return error")
	}
}

func TestGeneratePlainTextEmptyPaths(t *testing.T) {
	_, err := GeneratePlainText(nil)
	if err == nil {
		t.Error("GeneratePlainText(nil) should return error")
	}
}