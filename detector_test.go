package sigo

import (
	"bytes"
	"embed"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
)

//go:embed examples/*
var examples embed.FS

func TestDetect(t *testing.T) {
	exampleFiles, _ := examples.ReadDir("examples")
	for _, exampleFile := range exampleFiles {
		exampleFile := exampleFile
		if exampleFile.IsDir() {
			continue
		}
		t.Run(exampleFile.Name(), func(t *testing.T) {
			wantExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(exampleFile.Name())), ".")
			if wantExt == "svg" {
				t.Skip("svg is a text format, it has no binary signature")
			}
			f, err := examples.Open(filepath.Join("examples", exampleFile.Name()))
			if err != nil {
				t.Error(err)
				return
			}
			got, err := Detect(f)
			if err != nil {
				t.Errorf("unexpected err:= %v", err)
				return
			}
			gotExts := strings.Split(strings.ToLower(got.Extension), "|")
			for _, gotExt := range gotExts {
				if gotExt == wantExt {
					return
				}
			}
			t.Errorf("want %s, got %v", wantExt, gotExts)
		})
	}
}

func TestDetectBytes(t *testing.T) {
	ole := []byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1}
	xls := make([]byte, 520)
	copy(xls, ole)
	copy(xls[512:], []byte{0x9, 0x8, 0x10, 0x0, 0x0, 0x6, 0x5, 0x0})

	tests := []struct {
		name    string
		input   []byte
		want    string
		not     string
		wantErr error
	}{
		{
			name:  "longest wins",
			input: []byte{0x50, 0x4b, 0x3, 0x4, 0x14, 0x0, 0x6, 0x0, 0x0},
			want:  "MS Office 2007 documents",
		},
		{
			name:  "child wins over parent",
			input: xls,
			want:  "Excel spreadsheet subheader_1",
		},
		{
			name:  "offset 8",
			input: []byte("RIFF\x00\x00\x00\x00WEBPVP8 "),
			want:  "RIFF WebP",
		},
		{
			name:  "offset 30",
			input: append(append([]byte("PK\x03\x04"), make([]byte, 26)...), "mimetypeapplication/vnd.oasis.opendocument.spreadsheet"...),
			want:  "OpenDocument Spreadsheet",
		},
		{
			name:  "child without parent",
			input: append(make([]byte, 30), "mimetypeapplication/vnd.oasis.opendocument.spreadsheet"...),
			not:   "OpenDocument Spreadsheet",
		},
		{
			name:  "fallback",
			input: append(append([]byte("dir/a.txt"), make([]byte, 248)...), "ustar"...),
			want:  "Tape Archive",
		},
		{
			name:  "start wins over fallback",
			input: append(append([]byte("%PDF-1.4"), make([]byte, 249)...), "ustar"...),
			want:  "PDF file",
		},
		{
			name:  "merged duplicates",
			input: ole,
			want:  "Microsoft Office document",
		},
		{
			name:    "empty",
			input:   nil,
			wantErr: ErrUnknown,
		},
		{
			name:    "unknown",
			input:   []byte("zzzzzzzz"),
			wantErr: ErrUnknown,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// one byte per read, so signatures are split across reads
			got, err := Detect(iotest.OneByteReader(bytes.NewReader(tt.input)))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want err %v, got %v", tt.wantErr, err)
			}
			if tt.wantErr != nil {
				return
			}
			if tt.not != "" {
				if contains(strings.Split(got.Description, "|"), tt.not) {
					t.Errorf("want not %q", tt.not)
				}
				return
			}
			if !contains(strings.Split(got.Description, "|"), tt.want) {
				t.Errorf("want %q, got %q", tt.want, got.Description)
			}
		})
	}
}

type emptyReader struct{}

func (emptyReader) Read([]byte) (int, error) { return 0, nil }

func TestDetectNoProgress(t *testing.T) {
	if _, err := Detect(emptyReader{}); !errors.Is(err, io.ErrNoProgress) {
		t.Errorf("want %v, got %v", io.ErrNoProgress, err)
	}
}

func TestMerge(t *testing.T) {
	got := merge(
		&Signature{Description: "A", Extension: "X|Y"},
		&Signature{Description: "B", Extension: "Y|Z"},
	)
	if got.Description != "A|B" || got.Extension != "X|Y|Z" {
		t.Errorf("got %q, %q", got.Description, got.Extension)
	}
}

func TestMaxCursors(t *testing.T) {
	if n := len(roots) + maxChildren; n > maxCursors {
		t.Errorf("%d cursors do not fit maxCursors %d", n, maxCursors)
	}
}

func TestParents(t *testing.T) {
	for _, s := range sigs {
		if s.After == nil {
			continue
		}
		found := false
		for _, p := range sigs {
			if p.Offset == 0 && p.After == nil && bytes.Equal(p.Signature, s.After) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s: no parent %x at offset 0", s.Description, s.After)
		}
	}
}

func TestDetectHead(t *testing.T) {
	data, err := examples.ReadFile(filepath.Join("examples", "file_example_PNG_500kB.png"))
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 0, 2*readSize)
	got, head, err := DetectHead(iotest.HalfReader(bytes.NewReader(data)), buf)
	if err != nil {
		t.Fatal(err)
	}
	if got.Description != "PNG image" {
		t.Errorf("want PNG image, got %q", got.Description)
	}
	if !bytes.Equal(head, data[:len(head)]) {
		t.Error("head is not the start of the stream")
	}
	if &head[0] != &buf[:1][0] {
		t.Error("head is not reused")
	}
}

// BenchmarkDetect reports scannedB/op, the bytes scanned before the result was known,
// and readB/op, the bytes taken from the reader
func BenchmarkDetect(b *testing.B) {
	exampleFiles, _ := examples.ReadDir("examples")
	for _, exampleFile := range exampleFiles {
		data, err := examples.ReadFile(filepath.Join("examples", exampleFile.Name()))
		if err != nil {
			b.Fatal(err)
		}
		b.Run(exampleFile.Name(), func(b *testing.B) {
			var br bytes.Reader
			buf := make([]byte, 0, 2*readSize)
			var head []byte
			var scanned int
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				br.Reset(data)
				_, head, scanned, _ = detect(&br, buf)
			}
			b.ReportMetric(float64(scanned), "scannedB/op")
			b.ReportMetric(float64(len(head)), "readB/op")
		})
	}
}
