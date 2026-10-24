package sigo

import (
	"embed"
	_ "embed"
	"log"
	"path/filepath"
	"strings"
	"testing"
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
			switch wantExt {
			case "odt", "ods", "odp":
			default:
				// t.SkipNow()
			}
			log.Printf("%s: %s", wantExt, exampleFile.Name())
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

	//tests := []struct {
	//	name    string
	//	input   []byte
	//	want    *Signature
	//	wantErr bool
	//}{
	//	{
	//		name:  "ico",
	//		input: testICO,
	//		want: &Signature{
	//			Description: "Windows icon|printer spool file",
	//			Extension:   "ICO|SPL",
	//			Offset:      0,
	//			Signature:   []uint8{0x0, 0x0, 0x1, 0x0},
	//		},
	//		wantErr: false,
	//	},
	//}
	//for _, tt := range tests {
	//	tt := tt
	//	t.Run(tt.name, func(t *testing.T) {
	//		got, err := Detect(bytes.NewReader(tt.input))
	//		if (err != nil) != tt.wantErr {
	//			t.Errorf("Detect() error = %v, wantErr %v", err, tt.wantErr)
	//			return
	//		}
	//		if !reflect.DeepEqual(got, tt.want) {
	//			t.Errorf("Detect() got = %v, want %v", got, tt.want)
	//		}
	//	})
	//}
}
