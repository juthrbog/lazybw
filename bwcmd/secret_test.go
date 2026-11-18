package bwcmd

import (
	"bytes"
	"context"
	"testing"
)

func TestSecretBufferZerosOldArrayOnGrow(t *testing.T) {
	var s secretBuffer
	_, _ = s.Write([]byte("secret-part-one"))
	old := s.b[:cap(s.b)]
	_, _ = s.Write(bytes.Repeat([]byte("x"), 4*cap(s.b)))
	if &old[0] == &s.b[0] {
		t.Fatal("the buffer did not grow")
	}
	for i, b := range old {
		if b != 0 {
			t.Fatalf("old array byte %d = %q, want 0", i, b)
		}
	}
	if !bytes.HasPrefix(s.Bytes(), []byte("secret-part-one")) {
		t.Error("the grown buffer lost data")
	}
}

func TestItemsResultZerosOutput(t *testing.T) {
	for _, raw := range []string{
		`[{"id":"1","type":1,"name":"a","login":{"password":"pw"},"passwordHistory":[{"password":"old"}]}]`,
		`not json`,
	} {
		out := []byte(raw)
		itemsResult(out, 1)
		if !bytes.Equal(out, make([]byte, len(out))) {
			t.Errorf("output %q was not zeroed", raw)
		}
	}
}

func TestFetchTrashZerosOutput(t *testing.T) {
	var out []byte
	useRunner(t, func(context.Context, []string, []byte, ...string) ([]byte, []byte, error) {
		out = []byte(`[{"id":"1","type":1,"name":"a","login":{"password":"pw"},"passwordHistory":[{"password":"old"}]}]`)
		return out, nil, nil
	})
	res := FetchTrash("tok", 1, 1)().(TrashResult)
	if res.Err != nil || len(res.Items) != 1 {
		t.Fatalf("FetchTrash = %+v, want one item", res)
	}
	if !bytes.Equal(out, make([]byte, len(out))) {
		t.Errorf("output %q was not zeroed", out)
	}
}
