package generators

import (
	stripe "github.com/stripe/stripe-go/v79"
	"testing"
)

// Sample types for testing
type SampleStruct struct{}
type SampleStructV1 struct{}
type SampleStructV2 struct{}
type SampleStructHyphen struct{}
type SampleStructPtr *SampleStruct

// Unit tests
func TestGetTypePkg(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		wantPkg  string
		wantName string
	}{
		{
			name:     "String type",
			input:    "test string",
			wantPkg:  "string",
			wantName: "",
		},
		{
			name:     "Int64 type",
			input:    int64(42),
			wantPkg:  "int64",
			wantName: "",
		},
		{
			name:     "Slice of strings",
			input:    []string{"a", "b", "c"},
			wantPkg:  "[]string",
			wantName: "",
		},
		{
			name:     "Struct without version",
			input:    SampleStruct{},
			wantPkg:  "",
			wantName: "main",
		},
		{
			name:     "Struct with version suffix",
			input:    struct{ _ SampleStructV1 }{},
			wantPkg:  "",
			wantName: "main",
		},
		{
			name:     "Struct with hyphen in package name",
			input:    struct{ _ SampleStructHyphen }{},
			wantPkg:  "",
			wantName: "",
		},
		{
			name:     "Pointer to struct",
			input:    &SampleStruct{},
			wantPkg:  "github.com/Seann-Moser/go-serve/pkg/generator/generators",
			wantName: "generators",
		},
		{
			name:     "Array of structs",
			input:    []SampleStruct{{}, {}},
			wantPkg:  "github.com/Seann-Moser/go-serve/pkg/generator/generators",
			wantName: "generators",
		},
		{
			name:     "Package path with version",
			input:    struct{}{},
			wantPkg:  "github.com/stripe/stripe-go",
			wantName: "stripe",
		},
		{
			name:     "Package path with hyphen",
			input:    struct{}{},
			wantPkg:  "github.com/example/stripe-go",
			wantName: "stripe",
		},
		{
			name:     "Stripe invoice",
			input:    stripe.Invoice{},
			wantPkg:  "github.com/stripe/stripe-go/v79",
			wantName: "stripe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPkg, gotName := getTypePkg(tt.input)
			if gotPkg != tt.wantPkg {
				t.Errorf("getTypePkg() gotPkg = %v, want %v", gotPkg, tt.wantPkg)
			}
			if gotName != tt.wantName {
				t.Errorf("getTypePkg() gotName = %v, want %v", gotName, tt.wantName)
			}
		})
	}
}
