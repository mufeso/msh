package runbook

import (
	"strings"
	"testing"

	"example.com/msh/internal/decode"
)

func TestDecodeRunbook(t *testing.T) {
	in := `description: Restart the web tier
steps:
  - name: check disk
    task: disk-check
  - name: restart
    task: restart-service
    params: {service: nginx}
  - name: verify
    command: systemctl is-active nginx
    retries: 3
    retry_delay: 5s
`
	var rb Runbook
	if err := decode.Reader(strings.NewReader(in), decode.YAML, &rb); err != nil {
		t.Fatal(err)
	}
	if len(rb.Steps) != 3 || rb.Steps[1].Params["service"] != "nginx" {
		t.Fatalf("runbook = %+v", rb)
	}
	verify := rb.Steps[2]
	if verify.Retries != 3 || verify.RetryDelay == nil || verify.RetryDelay.String() != "5s" {
		t.Errorf("verify = %+v", verify)
	}
}
