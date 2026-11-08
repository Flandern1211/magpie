package gui

import (
	"testing"
	"time"

	"github.com/yetone/magpie/internal/sessions"
)

// A request still answering has no reply id: the message of its turn names
// its branch, as that branch's own file has it, and only when one session
// has it. A message two sessions share names neither, and the opening's
// session stands, as before.
func TestDshLiveRequestIsNamedByItsTurn(t *testing.T) {
	source := sessions.Session{Agent: "dsh", ID: "source", Convs: []string{"magpie-open"}, TurnConvs: []string{"magpie-msg-source"}}
	fork := sessions.Session{Agent: "dsh", ID: "fork", SeedConvs: []string{"magpie-open"}, TurnConvs: []string{"magpie-msg-fork"}}
	list := []sessions.Session{source, fork}
	now := time.Now()
	routes := []ctxRec{
		{Agent: "dsh", Conv: "magpie-open", TurnConvs: []string{"magpie-msg-fork"}, Time: now, End: now},   // in flight in the fork
		{Agent: "dsh", Conv: "magpie-open", TurnConvs: []string{"magpie-msg-source"}, Time: now, End: now}, // in flight in the source
		{Agent: "dsh", Conv: "magpie-open", TurnConvs: []string{"magpie-msg-else"}, Time: now, End: now},   // a message no session has
	}
	joinConvs(routes, func() []sessions.Session { return list })
	for i, want := range []string{"fork", "source", "source"} {
		if routes[i].Session != want || routes[i].Unmatched {
			t.Errorf("route %d: session %q, unmatched %v; want %q", i, routes[i].Session, routes[i].Unmatched, want)
		}
	}
	// a message both sessions have names neither: the opening's, the source's
	source.TurnConvs = append(source.TurnConvs, "magpie-msg-both")
	fork.TurnConvs = append(fork.TurnConvs, "magpie-msg-both")
	list = []sessions.Session{source, fork}
	both := []ctxRec{{Agent: "dsh", Conv: "magpie-open", TurnConvs: []string{"magpie-msg-both"}, Time: now, End: now}}
	joinConvs(both, func() []sessions.Session { return list })
	if both[0].Session != "source" || both[0].Unmatched {
		t.Errorf("a shared message: session %q, unmatched %v; want the opening's, the source's", both[0].Session, both[0].Unmatched)
	}
}
