package commands

import "testing"

func TestParseTableName(t *testing.T) {
	tests := []struct {
		input string
		want  tableName
	}{
		{input: "users", want: tableName{Schema: "public", Name: "users"}},
		{input: "billing.invoices", want: tableName{Schema: "billing", Name: "invoices"}},
	}

	for _, test := range tests {
		got, err := parseTableName(test.input)
		if err != nil {
			t.Fatalf("parseTableName(%q) returned error: %v", test.input, err)
		}
		if got != test.want {
			t.Fatalf("parseTableName(%q) = %#v, want %#v", test.input, got, test.want)
		}
	}
}

func TestParseTableNameRejectsInvalidInput(t *testing.T) {
	invalid := []string{"", "bad-name", "too.many.parts", "1users"}
	for _, input := range invalid {
		if _, err := parseTableName(input); err == nil {
			t.Fatalf("parseTableName(%q) expected an error", input)
		}
	}
}

func TestQuoteIdentifiers(t *testing.T) {
	got := quoteIdentifiers([]string{"id", "user_name"})
	want := `"id", "user_name"`
	if got != want {
		t.Fatalf("quoteIdentifiers() = %q, want %q", got, want)
	}
}

func TestQuoteLiteral(t *testing.T) {
	got := quoteLiteral("don't")
	want := `'don''t'`
	if got != want {
		t.Fatalf("quoteLiteral() = %q, want %q", got, want)
	}
}

func TestParseIdentifierList(t *testing.T) {
	got, err := parseIdentifierList("id, name, created_at")
	if err != nil {
		t.Fatalf("parseIdentifierList returned error: %v", err)
	}
	want := []string{"id", "name", "created_at"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseIdentifierList()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
