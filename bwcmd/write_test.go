package bwcmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// TestCreateItemSendsItemOnStdin checks that CreateItem sends the item as
// base64-encoded JSON on stdin, and that no field value is in the args.
func TestCreateItemSendsItemOnStdin(t *testing.T) {
	var gotArgs []string
	var gotStdin []byte
	var gotToken string
	useRunner(t, func(_ context.Context, env []string, stdin []byte, args ...string) ([]byte, []byte, error) {
		gotArgs = args
		gotStdin = append([]byte(nil), stdin...)
		gotToken, _ = envValue(env, "BW_SESSION")
		return []byte(`{"id":"new-id","name":"Gmail","login":{"password":"hunter2"}}`), nil, nil
	})

	item := NewItemJSON(ItemTypeLogin)
	item["name"] = "Gmail"
	item["login"].(map[string]any)["password"] = "hunter2"
	res := CreateItem("session-token", 4, item, []string{"Gmail", "hunter2"})().(CreateResult)

	if res.Err != nil || res.ID != "new-id" || res.Epoch != 4 {
		t.Fatalf("CreateItem = %+v, want ID new-id and epoch 4", res)
	}
	if want := []string{"create", "item"}; !reflect.DeepEqual(gotArgs, want) {
		t.Errorf("args = %q, want %q", gotArgs, want)
	}
	for _, a := range gotArgs {
		if strings.Contains(a, "hunter2") || strings.Contains(a, "Gmail") {
			t.Errorf("args %q hold a field value", gotArgs)
		}
	}
	if gotToken != "session-token" {
		t.Errorf("BW_SESSION = %q, want the token", gotToken)
	}
	raw, err := base64.StdEncoding.DecodeString(string(gotStdin))
	if err != nil {
		t.Fatalf("stdin is not base64: %v", err)
	}
	var sent map[string]any
	if err := json.Unmarshal(raw, &sent); err != nil {
		t.Fatalf("stdin is not item JSON: %v", err)
	}
	if sent["name"] != "Gmail" || sent["type"] != float64(ItemTypeLogin) {
		t.Errorf("sent item = %v, want the login Gmail", sent)
	}
	if login, _ := sent["login"].(map[string]any); login["password"] != "hunter2" {
		t.Errorf("sent login = %v, want the password", sent["login"])
	}
}

// TestNewItemJSONDefaults checks the defaults of a new item.
func TestNewItemJSONDefaults(t *testing.T) {
	for _, typ := range []ItemType{ItemTypeLogin, ItemTypeSecureNote} {
		item := NewItemJSON(typ)
		for k, want := range map[string]any{
			"organizationId": nil, "folderId": nil, "favorite": false, "reprompt": RepromptNone, "type": int(typ),
		} {
			if got := item[k]; got != want {
				t.Errorf("type %d: %s = %v, want %v", typ, k, got, want)
			}
		}
	}
	if NewItemJSON(ItemTypeLogin)["login"] == nil {
		t.Error("a new login has no login object")
	}
	if NewItemJSON(ItemTypeSecureNote)["secureNote"] == nil {
		t.Error("a new secure note has no secureNote object")
	}
}

// TestCreateItemFailureHidesValues checks that a failed create returns the
// bw error, and that the debug log shows no field value and no item JSON.
func TestCreateItemFailureHidesValues(t *testing.T) {
	buf := captureLog(t)
	var payload string
	useRunner(t, func(_ context.Context, _ []string, stdin []byte, _ ...string) ([]byte, []byte, error) {
		payload = string(stdin)
		return nil, []byte("Item hunter2 failed for " + payload + "\nName is required.\n"), errors.New("exit status 1")
	})

	item := NewItemJSON(ItemTypeLogin)
	item["login"].(map[string]any)["password"] = "hunter2"
	res := CreateItem("session-token", 1, item, []string{"hunter2"})().(CreateResult)

	var bwErr *Error
	if !errors.As(res.Err, &bwErr) || bwErr.Msg != "Name is required." {
		t.Errorf("err = %#v, want the last stderr line", res.Err)
	}
	logged := buf.String()
	if !strings.Contains(logged, "bw create item failed") {
		t.Errorf("log = %q, want the subcommand", logged)
	}
	for _, secret := range []string{"hunter2", payload, "session-token"} {
		if strings.Contains(logged, secret) {
			t.Errorf("log shows %q:\n%s", secret, logged)
		}
	}
}

// TestCreateItemNeedsID checks that a create that prints no item ID fails.
func TestCreateItemNeedsID(t *testing.T) {
	useRunner(t, func(context.Context, []string, []byte, ...string) ([]byte, []byte, error) {
		return []byte(`{}`), nil, nil
	})
	if res := CreateItem("t", 1, NewItemJSON(ItemTypeSecureNote), nil)().(CreateResult); res.Err == nil {
		t.Errorf("CreateItem = %+v, want an error", res)
	}
}

// TestGetItemKeepsNumbers checks that GetItem keeps each key of the item
// and keeps the numbers as bw printed them.
func TestGetItemKeepsNumbers(t *testing.T) {
	var gotArgs []string
	useRunner(t, func(_ context.Context, _ []string, _ []byte, args ...string) ([]byte, []byte, error) {
		gotArgs = args
		return []byte(`{"id":"abc","type":1,"reprompt":0,"revisionDate":"2026-01-01T00:00:00Z","passwordHistory":[{"password":"old"}],"login":{"uris":[{"match":3,"uri":"a"}]}}`), nil, nil
	})
	res := GetItem("t", 2, "abc")().(GetItemResult)
	if res.Err != nil || res.ID != "abc" || res.Epoch != 2 {
		t.Fatalf("GetItem = %+v", res)
	}
	if want := []string{"get", "item", "abc"}; !reflect.DeepEqual(gotArgs, want) {
		t.Errorf("args = %q, want %q", gotArgs, want)
	}
	if res.Item["type"] != json.Number("1") {
		t.Errorf("type = %#v, want json.Number 1", res.Item["type"])
	}
	raw, err := json.Marshal(res.Item)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"match":3`, `"passwordHistory":[{"password":"old"}]`, `"revisionDate":"2026-01-01T00:00:00Z"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("encoded item %s does not hold %s", raw, want)
		}
	}
}

// TestEditItemSendsItemOnStdin checks that EditItem names the item in the
// args and sends the item JSON only on stdin.
func TestEditItemSendsItemOnStdin(t *testing.T) {
	var gotArgs []string
	var gotStdin []byte
	useRunner(t, func(_ context.Context, _ []string, stdin []byte, args ...string) ([]byte, []byte, error) {
		gotArgs, gotStdin = args, append([]byte(nil), stdin...)
		return []byte(`{"id":"abc"}`), nil, nil
	})
	item := map[string]any{"id": "abc", "name": "Gmail", "login": map[string]any{"password": "hunter2"}}
	res := EditItem("t", 5, "abc", item, []string{"hunter2"})().(EditResult)
	if res.Err != nil || res.ID != "abc" || res.Epoch != 5 {
		t.Fatalf("EditItem = %+v", res)
	}
	if want := []string{"edit", "item", "abc"}; !reflect.DeepEqual(gotArgs, want) {
		t.Errorf("args = %q, want %q", gotArgs, want)
	}
	for _, a := range gotArgs {
		if strings.Contains(a, "hunter2") {
			t.Errorf("args %q hold a field value", gotArgs)
		}
	}
	raw, err := base64.StdEncoding.DecodeString(string(gotStdin))
	if err != nil || !strings.Contains(string(raw), `"password":"hunter2"`) {
		t.Errorf("stdin = %q (%v), want the encoded item", raw, err)
	}
}
