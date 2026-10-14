package models

import "testing"

func TestSanitizeOrder(t *testing.T) {
	tests := []struct{ in, want string }{
		{"id desc", "id desc"},
		{"a_id asc", "a_id asc"},
		{"id desc, q_date", "id desc, q_date"},
		{" p_order ,p_id DESC ", "p_order, p_id DESC"},
		{"RAND()", "RAND()"},
		{"", ""},

		// 컬럼 나열이 아닌 것은 통째로 버린다.
		{"(select 1)", ""},
		{"a_id; drop table answers_tb", ""},
		{"a_id desc, (select sleep(5))", ""},
		{"if(1=1,a_id,a_date)", ""},
		{"a_id -- x", ""},
		{"a_id desc limit 1", ""},
	}

	for _, tt := range tests {
		if got := Ordering(tt.in).Order; got != tt.want {
			t.Errorf("Ordering(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
