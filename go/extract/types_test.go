package extract

import "testing"

func TestExtractLLTag(t *testing.T) {
	tests := []struct {
		raw     string
		wantKey string
		wantOk  bool
	}{
		{raw: `ll:"Object"`, wantKey: "Object", wantOk: true},
		{raw: `ll:"Value"`, wantKey: "Value", wantOk: true},
		{raw: `json:"foo" ll:"Bar"`, wantKey: "Bar", wantOk: true},
		{raw: `json:"foo"`, wantKey: "", wantOk: false},
		{raw: ``, wantKey: "", wantOk: false},
		{raw: `ll:"-"`, wantKey: "", wantOk: false},
	}
	for _, tt := range tests {
		key, ok := extractLLTag(tt.raw)
		if key != tt.wantKey || ok != tt.wantOk {
			t.Errorf("extractLLTag(%q) = (%q, %v), want (%q, %v)",
				tt.raw, key, ok, tt.wantKey, tt.wantOk)
		}
	}
}
