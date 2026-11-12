package util_test

import (
	"strings"
	"testing"

	"github.com/ngbcoder/Unseal/internal/util"
)

func TestHTMLToText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain text passes through",
			in:   "Just a summary.",
			want: "Just a summary.",
		},
		{
			name: "paragraphs become separate lines",
			in:   "<p>First paragraph.</p><p>Second paragraph.</p>",
			want: "First paragraph.\r\n\r\nSecond paragraph.",
		},
		{
			name: "a div with text of its own becomes a block",
			in:   "<div>Block text</div>",
			want: "Block text",
		},
		{
			name: "a div holding only nested elements gets no break",
			in:   "<div><span>Nested</span></div>",
			want: "Nested",
		},
		{
			name: "span is unwrapped",
			in:   "<span>Spanned</span>",
			want: "Spanned",
		},
		{
			name: "label is unwrapped",
			in:   "<label>Labelled</label>",
			want: "Labelled",
		},
		{
			name: "br becomes a line break",
			in:   "First<br>Second",
			want: "First\r\nSecond",
		},
		{
			name: "script content is dropped",
			in:   "<p>Kept</p><script>alert('gone')</script>",
			want: "Kept",
		},
		{
			name: "style content is dropped",
			in:   "<style>.x{color:red}</style><p>Kept</p>",
			want: "Kept",
		},
		{
			name: "head content is dropped",
			in:   "<html><head><title>Dropped</title></head><body><p>Kept</p></body></html>",
			want: "Kept",
		},
		{
			name: "comments are dropped",
			in:   "<p>Kept</p><!-- a comment -->",
			want: "Kept",
		},
		{
			name: "newlines in the source do not break the block logic",
			in:   "<p>First line\nsecond line</p><p>Second</p>",
			want: "First line second line\r\n\r\nSecond",
		},
		{
			name: "entities are decoded",
			in:   "<p>Tom &amp; Jerry</p>",
			want: "Tom & Jerry",
		},
		{
			name: "empty input gives empty output",
			in:   "",
			want: "",
		},
		{
			name: "whitespace only input gives empty output",
			in:   "   \n\t ",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := util.HTMLToText(tt.in)
			if got != tt.want {
				t.Errorf("HTMLToText(%q)\n got %q\nwant %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestHTMLToTextKeepsNestedText(t *testing.T) {
	in := `<div><p>Chapter one</p><p>Chapter two</p></div>`

	got := util.HTMLToText(in)

	if !strings.Contains(got, "Chapter one") || !strings.Contains(got, "Chapter two") {
		t.Errorf("nested text was lost: %q", got)
	}
}

func TestHTMLToTextDoesNotLeakScriptSource(t *testing.T) {
	in := `<p>Summary</p><script>var secret = "hunter2";</script>`

	got := util.HTMLToText(in)

	if strings.Contains(got, "hunter2") {
		t.Errorf("script source leaked into the output: %q", got)
	}
}
