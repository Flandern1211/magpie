package agent

import (
	"os"
	"strings"
	"testing"
)

// A dsh fork's own session id reaches magpie only when dsh sends pi-ai's
// session affinity: magpie's routes ask for it, so Routing can tell the
// fork from the session it copied.
func TestDshRouteSendsSessionAffinityHeaders(t *testing.T) {
	home, _, web := dshRouteHome(t)
	a := dsh(home)
	if err := a.Field("model").Set("magpie/deepseek/pro"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(web)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "sendSessionAffinityHeaders: true") {
		t.Fatalf("magpie's dsh route asks for no session affinity:\n%s", b)
	}
}
