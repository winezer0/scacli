package report

import "testing"

func TestResolveFormats(t *testing.T) {
	tests := []struct {
		name    string
		values  []string
		engine  string
		want    []Format
		wantErr bool
	}{
		{name: "MurphySec default", engine: "murphysec", want: []Format{FormatJSON}},
		{name: "OpenSCA default", engine: "opensca", want: []Format{FormatHTML}},
		{name: "OSV default", engine: "osv-scanner", want: []Format{FormatJSON, FormatHTML, FormatMarkdown}},
		{name: "comma separated", values: []string{"json,html", "md"}, engine: "osv-scanner", want: []Format{FormatJSON, FormatHTML, FormatMarkdown}},
		{name: "unsupported filtered", values: []string{"json,html"}, engine: "opensca", want: []Format{FormatHTML}},
		{name: "unsupported only", values: []string{"json"}, engine: "opensca", wantErr: true},
		{name: "invalid format", values: []string{"xml"}, engine: "osv-scanner", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveFormats(test.values, test.engine)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, test.wantErr)
			}
			if test.wantErr {
				return
			}
			if len(got) != len(test.want) {
				t.Fatalf("formats = %v, want %v", got, test.want)
			}
			for index := range test.want {
				if got[index] != test.want[index] {
					t.Fatalf("formats = %v, want %v", got, test.want)
				}
			}
		})
	}
}
