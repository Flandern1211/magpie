package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// dsh's requests carry no session id, and a fork copies its source's
// opening: the newest user message is what tells a request still answering
// which branch it is of, hashed as the branch's own session file has it.
func TestDshLastConvsHashTheNewestUserMessage(t *testing.T) {
	body := []byte(`{"model":"m","messages":[` +
		`{"role":"system","content":"s"},` +
		`{"role":"user","content":"old"},` +
		`{"role":"assistant","content":"a"},` +
		`{"role":"user","content":[{"type":"text","text":"new"}]}]}`)
	got := dshLastConvs("dsh", body)
	if len(got) != 3 {
		t.Fatalf("ids %v, want the newest message's three spellings", got)
	}
	for i, want := range []string{
		`{"role":"user","content":"new"}`,
		`{"role":"user","content":[{"type":"text","text":"new"}]}`,
		`{"role":"user","content":[{"type":"text","text":"new","cache_control":{"type":"ephemeral"}}]}`,
	} {
		sum := sha256.Sum256([]byte(want))
		if id := "magpie-" + hex.EncodeToString(sum[:12]); got[i] != id {
			t.Errorf("spelling %d: %q, want %q", i, got[i], id)
		}
	}
}

// Only dsh's requests are read that way, and only when the newest turn is a
// message of one text part.
func TestDshLastConvsOnlyForDshWithATextMessage(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":"new"}]}`)
	if got := dshLastConvs("pi", body); got != nil {
		t.Errorf("another agent: %v, want none", got)
	}
	if got := dshLastConvs("dsh", []byte(`{"messages":[{"role":"assistant","content":"a"}]}`)); got != nil {
		t.Errorf("no user message: %v, want none", got)
	}
	if got := dshLastConvs("dsh", []byte(`{"messages":[{"role":"user","content":[{"type":"image","url":"x"}]}]}`)); got != nil {
		t.Errorf("no text part: %v, want none", got)
	}
}
