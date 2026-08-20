package systemd

import "testing"

type fakeRunner struct {
	showOut string
}

func (f *fakeRunner) Run(_ string, _ ...string) (string, error) {
	return f.showOut, nil
}

// TestExecStarts_MultipleExecStartEntries reproduces the desync observed on
// a real host: `systemctl show` emits one ExecStart= line per directive, so
// a unit with two ExecStart= lines (legal and common for oneshot-style
// units) used to shift every later unit's path by one line. Confirmed
// against a real host with 190 units producing 359 ExecStart lines.
func TestExecStarts_MultipleExecStartEntries(t *testing.T) {
	showOut := "Id=a.service\n" +
		"ExecStart={ path=/usr/bin/a1 ; argv[]=/usr/bin/a1 }\n" +
		"ExecStart={ path=/usr/bin/a2 ; argv[]=/usr/bin/a2 }\n" +
		"\n" +
		"Id=b.service\n" +
		"ExecStart={ path=/usr/bin/b ; argv[]=/usr/bin/b }\n" +
		"\n" +
		"Id=c.service\n" +
		"ExecStart={ path=/usr/bin/c ; argv[]=/usr/bin/c }\n"

	s := SystemD{runner: &fakeRunner{showOut: showOut}}
	paths := s.execStarts([]string{"a.service", "b.service", "c.service"})

	want := map[string]string{
		"a.service": "/usr/bin/a1",
		"b.service": "/usr/bin/b",
		"c.service": "/usr/bin/c",
	}
	for name, path := range want {
		if paths[name] != path {
			t.Errorf("execStarts()[%q] = %q, want %q", name, paths[name], path)
		}
	}
}

func TestExecStarts_NoExecStartEntry(t *testing.T) {
	showOut := "Id=a.service\n" +
		"ExecStart=\n" +
		"\n" +
		"Id=b.service\n" +
		"ExecStart={ path=/usr/bin/b ; argv[]=/usr/bin/b }\n"

	s := SystemD{runner: &fakeRunner{showOut: showOut}}
	paths := s.execStarts([]string{"a.service", "b.service"})

	if paths["a.service"] != "" {
		t.Errorf("execStarts()[a.service] = %q, want empty", paths["a.service"])
	}
	if paths["b.service"] != "/usr/bin/b" {
		t.Errorf("execStarts()[b.service] = %q, want /usr/bin/b", paths["b.service"])
	}
}
