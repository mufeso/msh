package app

import (
	"encoding/csv"
	"strings"
	"testing"

	"example.com/msh/internal/exitcode"
)

func TestExecCSV(t *testing.T) {
	fleet := newTestFleet(t, 3, nil)
	stdout, stderr, code := fleet.run(t, []string{"exec"}, "--all", "--output", "csv", "--command", "echo 'a,b'")
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	records, err := csv.NewReader(strings.NewReader(stdout)).ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v\n%s", err, stdout)
	}
	if len(records) != 4 || records[0][0] != "server" {
		t.Fatalf("records = %q", records)
	}
	for _, r := range records[1:] {
		if r[2] != "ok" || r[7] != "a,b\n" {
			t.Errorf("row = %q", r)
		}
	}
}

func TestHostsListCSV(t *testing.T) {
	stdout, _, code := runCLI(t, "hosts", "list", "--inventory", "testdata/inventory.yaml", "--output", "csv")
	if code != exitcode.OK {
		t.Fatalf("code = %v", code)
	}
	if !strings.HasPrefix(stdout, "NAME,ADDRESS,PORT,USER,GROUPS,VIA\nweb01,192.0.2.10,22,admin,\"web,production\",direct\n") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestCSVRejectedForSingleObjects(t *testing.T) {
	_, stderr, code := runCLI(t, "config", "validate", "--inventory", "testdata/inventory.yaml", "--output", "csv")
	if code != exitcode.Usage || !contains(stderr, "not supported") {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}
