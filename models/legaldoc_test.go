package models

import "testing"

func TestLegaldocValidate(t *testing.T) {
	valid := func() Legaldoc {
		return Legaldoc{Project: 1, Type: "privacy", Lang: "ko", Title: "개인정보 처리방침", Body: "본문", Effectivedate: "2026-11-10"}
	}

	tests := []struct {
		name   string
		mutate func(d *Legaldoc)
		ok     bool
	}{
		{"valid", func(d *Legaldoc) {}, true},
		{"hyphenated type", func(d *Legaldoc) { d.Type = "location-terms" }, true},
		{"normalized type and lang", func(d *Legaldoc) { d.Type, d.Lang = " Privacy ", "KO" }, true},
		{"empty lang defaults to ko", func(d *Legaldoc) { d.Lang = "" }, true},

		{"no project", func(d *Legaldoc) { d.Project = 0 }, false},
		// 종류는 URL 경로 조각이 된다. 경로를 벗어나는 문자는 받지 않는다.
		{"type with slash", func(d *Legaldoc) { d.Type = "a/b" }, false},
		{"type with dots", func(d *Legaldoc) { d.Type = ".." }, false},
		{"type with space", func(d *Legaldoc) { d.Type = "privacy policy" }, false},
		{"empty type", func(d *Legaldoc) { d.Type = "" }, false},
		{"long type", func(d *Legaldoc) { d.Type = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" }, false},
		{"bad lang", func(d *Legaldoc) { d.Lang = "kor" }, false},
		{"empty title", func(d *Legaldoc) { d.Title = "  " }, false},
		{"blank body", func(d *Legaldoc) { d.Body = " \n " }, false},
		{"bad date format", func(d *Legaldoc) { d.Effectivedate = "2026/11/10" }, false},
		{"impossible date", func(d *Legaldoc) { d.Effectivedate = "2026-02-30" }, false},
		{"date with time", func(d *Legaldoc) { d.Effectivedate = "2026-11-10 00:00:00" }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := valid()
			tt.mutate(&d)
			d.Normalize()
			err := d.Validate()
			if (err == nil) != tt.ok {
				t.Errorf("Validate() error = %v, want ok=%v", err, tt.ok)
			}
		})
	}
}
