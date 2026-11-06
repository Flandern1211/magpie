package sessions

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
)

// dshReply is an assistant/message line holding a reply's replay state, as
// dsh writes it: the id is the one the gateway put on the reply.
func dshReplyWithID(seq int, id string) string {
	return `{"type":"assistant/message","seq":` + strconv.Itoa(seq) + `,"time":1000,"data":{"message":{"source":{"provider":"magpie","model":"group/flash","replayState":{"response":{"kind":"pi-ai","version":2,"api":"openai-completions","provider":"magpie","model":"group/flash","responseId":"` + id + `","stopReason":"stop"}}}}}}`
}

// dshSessionFile writes a session file with the lines given and parses it.
func dshSessionFile(t *testing.T, lines []string) *state {
	t.Helper()
	p := filepath.Join(t.TempDir(), "session.jsonl")
	b := ""
	for _, l := range lines {
		b += l + "\n"
	}
	if err := os.WriteFile(p, []byte(b), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return parseDsh(file{agent: "dsh", path: p, main: true, size: st.Size(), mod: st.ModTime()}, nil)
}

func TestDshResponseIDsSkipTheSeed(t *testing.T) {
	head := `{"type":"session","version":4,"id":"session-child","cwd":"/w","createdAt":1000,"delegationDepth":0,"isSeeded":true}`
	s := dshSessionFile(t, []string{
		head,
		dshReplyWithID(1, "inherited-1"),
		dshReplyWithID(2, "inherited-2"),
		`{"type":"session/end-seed","seq":3,"time":1000,"data":{"inherited":true}}`,
		dshReplyWithID(4, "own-1"),
		dshReplyWithID(5, "own-2"),
	})
	if want := []string{"own-1", "own-2"}; !slices.Equal(s.ResponseIDs, want) {
		t.Fatalf("fork's own replies: %v, want %v", s.ResponseIDs, want)
	}
}

// A file in the format before session/end-seed marks its inherited events by
// seq: those are its source's, not its own.
func TestDshResponseIDsSkipTheSeedLength(t *testing.T) {
	head := `{"type":"session","version":1,"id":"session-old","cwd":"/w","createdAt":1000,"seedLength":2,"isSeeded":true}`
	s := dshSessionFile(t, []string{
		head,
		dshReplyWithID(0, "inherited"),
		dshReplyWithID(1, "inherited-2"),
		dshReplyWithID(2, "own"),
	})
	if want := []string{"own"}; !slices.Equal(s.ResponseIDs, want) {
		t.Fatalf("fork's own replies: %v, want %v", s.ResponseIDs, want)
	}
}

// A session that is no fork owns every reply of its own.
func TestDshResponseIDsOfASession(t *testing.T) {
	head := `{"type":"session","version":4,"id":"session-plain","cwd":"/w","createdAt":1000,"delegationDepth":0,"isSeeded":false}`
	s := dshSessionFile(t, []string{
		head,
		dshReplyWithID(1, "a"),
		dshReplyWithID(2, "b"),
	})
	if want := []string{"a", "b"}; !slices.Equal(s.ResponseIDs, want) {
		t.Fatalf("replies: %v, want %v", s.ResponseIDs, want)
	}
}

// A reply dsh did not get from magpie's gateway is no route's to name: its id
// would match nothing, and is not kept.
func TestDshResponseIDsOnlyFromTheGateway(t *testing.T) {
	head := `{"type":"session","version":4,"id":"session-plain","cwd":"/w","createdAt":1000,"delegationDepth":0,"isSeeded":false}`
	s := dshSessionFile(t, []string{
		head,
		`{"type":"assistant/message","seq":1,"time":1000,"data":{"message":{"source":{"provider":"deepseek","model":"deepseek-v4","replayState":{"response":{"provider":"deepseek","responseId":"vendor-1"}}}}}}`,
		dshReplyWithID(2, "gw-1"),
	})
	if want := []string{"gw-1"}; !slices.Equal(s.ResponseIDs, want) {
		t.Fatalf("replies: %v, want %v", s.ResponseIDs, want)
	}
}

// A fork read before its seed's end lets its source's replies stand for it
// for that read; the marker, arriving later, takes them back (TestDshCalls-
// EndSeedAcrossReads reads the calls the same way).
func TestDshResponseIDsEndSeedAcrossReads(t *testing.T) {
	head := `{"type":"session","version":4,"id":"session-child","cwd":"/w","createdAt":1000,"delegationDepth":0,"isSeeded":true}`
	dir := t.TempDir()
	p := filepath.Join(dir, "session.jsonl")
	write := func(lines ...string) {
		f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		for _, l := range lines {
			if _, err := f.WriteString(l + "\n"); err != nil {
				t.Fatal(err)
			}
		}
	}
	f := func() file {
		st, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		return file{agent: "dsh", path: p, main: true, size: st.Size(), mod: st.ModTime()}
	}
	write(head, dshReplyWithID(1, "inherited"))
	first := parseDsh(f(), nil)
	if len(first.ResponseIDs) != 1 {
		t.Fatalf("before the marker: %v", first.ResponseIDs)
	}
	write(`{"type":"session/end-seed","seq":2,"time":1000,"data":{"inherited":true}}`, dshReplyWithID(3, "own"))
	next := parseDsh(f(), first)
	if want := []string{"own"}; !slices.Equal(next.ResponseIDs, want) {
		t.Fatalf("after the marker: %v, want %v", next.ResponseIDs, want)
	}
}
