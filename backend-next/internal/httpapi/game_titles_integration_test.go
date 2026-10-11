//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"net/http"
	"testing"
)

func TestGameTitlesBridgeRejectsPlayerClaimsAndOldSessions(t *testing.T) {
	f := newSocialFixtureV2(t)
	token := "title-local-test-node-token-0123456789"
	f.app.Config.Nodes = []config.Node{{ID: "amiya", Chat: true, TokenSHA256: store.Digest([]byte(token))}}
	uuid := "11111111-2222-3333-4444-555555555555"
	raw, _ := json.Marshal(map[string]any{"players": []map[string]string{{"playerUuid": uuid, "gameId": "NoAppAccount", "sessionEpoch": "epoch-2"}}})
	f.app.Core.Connect("amiya", func(context.Context, string, any) error { return nil })
	f.app.Core.SetPresence("amiya", raw)
	in := store.GameTitleAccess{PlayerUUID: uuid, SessionEpoch: "epoch-2", CatalogRevision: 1}
	call := func(auth, node string, input store.GameTitleAccess, want int) {
		t.Helper()
		b, _ := json.Marshal(input)
		r, _ := http.NewRequest("POST", f.http.URL+"/bridge/v1/game-chat/titles/state", bytes.NewReader(b))
		r.Header.Set("Authorization", "Bearer "+auth)
		r.Header.Set("X-Deuterium-Node-ID", node)
		r.Header.Set("Content-Type", "application/json")
		response, err := f.http.Client().Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var body map[string]any
		json.NewDecoder(response.Body).Decode(&body)
		if response.StatusCode != want {
			t.Fatalf("got %d want %d: %v", response.StatusCode, want, body)
		}
	}
	call(f.tokens["Alice"], "amiya", in, 401)
	call(token, "odyssey", in, 401)
	in.SessionEpoch = "epoch-1"
	call(token, "amiya", in, 401)
	in.SessionEpoch = "epoch-2"
	call(token, "amiya", in, 200)
	in.TitleID = store.DefaultTitleID
	call(token, "amiya", in, 400)
	in.TitleID = ""
	f.app.Core.Disconnect("amiya")
	call(token, "amiya", in, 401)
}

func TestGameTitlesCatalogAndPermissionHistory(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	f.request(t, "Bob", "GET", "/api/v1/admin/titles", nil, 403)
	if _, err := f.store.DB.Exec("INSERT INTO identity_permissions VALUES(?,'platform.admin')", f.users["Alice"].ID); err != nil {
		t.Fatal(err)
	}
	create := func(internal, display string) store.GameTitle {
		t.Helper()
		raw := socialDataV2(f.request(t, "Alice", "POST", "/api/v1/admin/titles", store.GameTitleChange{ClientRequestID: "create-" + internal, InternalID: internal, DisplayName: display, Weight: 10, Status: "PUBLISHED"}, 200))
		b, _ := json.Marshal(raw)
		var out store.GameTitle
		json.Unmarshal(b, &out)
		return out
	}
	a := create("first", "先行者")
	b := create("second", "#d9afd9参#c3bddc考#adcbde这#97d9e1样")
	catalog, err := f.store.GameTitles(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Items[0].ID != b.ID || catalog.Items[1].ID != a.ID {
		t.Fatal("later publication must sort first")
	}
	in := store.GameTitleAccess{PlayerUUID: f.users["Bob"].ServerUUID, CatalogRevision: catalog.Revision, AllowedIDs: []string{a.ID, b.ID}}
	state, err := f.store.ReconcileGameTitle(ctx, "Bob", in)
	if err != nil || state.TitleID != store.DefaultTitleID {
		t.Fatalf("default: %+v %v", state, err)
	}
	choose := func(title, request string) store.GameTitlePlayer {
		t.Helper()
		in.TitleID = title
		in.ClientRequestID = request
		in.ExpectedVersion = state.Version
		out, e := f.store.ReconcileGameTitle(ctx, "Bob", in)
		if e != nil {
			t.Fatal(e)
		}
		in.TitleID = ""
		return out
	}
	state = choose(a.ID, "choose-a")
	state = choose(b.ID, "choose-b")
	oldVersion := state.Version
	in.AllowedIDs = []string{a.ID}
	state, err = f.store.ReconcileGameTitle(ctx, "Bob", in)
	if err != nil || state.TitleID != a.ID || state.Version != oldVersion+1 {
		t.Fatalf("expiry fallback: %+v %v", state, err)
	}
	in.AllowedIDs = []string{a.ID, b.ID}
	state, err = f.store.ReconcileGameTitle(ctx, "Bob", in)
	if err != nil || state.TitleID != a.ID {
		t.Fatal("permission restoration must not re-equip expired title")
	}
	in.TitleID = b.ID
	in.ClientRequestID = "choose-b"
	in.ExpectedVersion = 2
	replay, err := f.store.ReconcileGameTitle(ctx, "Bob", in)
	if err != nil || replay.TitleID != a.ID {
		t.Fatalf("old retry rewore title: %+v %v", replay, err)
	}
	in.TitleID = ""
	// Same stable ID changes the projection of everyone wearing it.
	rename := store.GameTitleChange{ClientRequestID: "rename-a", ExpectedVersion: a.Version, InternalID: a.InternalID, DisplayName: "#112233更#445566名", Weight: a.Weight, Status: a.Status}
	raw, err := f.store.ChangeGameTitle(ctx, f.users["Alice"].ID, a.ID, rename)
	if err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(raw, &a)
	profiles, err := f.store.GameTitleProfiles(ctx)
	if err != nil || profiles[0].DisplayName != rename.DisplayName {
		t.Fatalf("rename propagation: %+v %v", profiles, err)
	}
	if _, err = f.store.ReconcileGameTitle(ctx, "Bob", in); !errors.Is(err, store.ErrConflict) {
		t.Fatal("stale permission catalog must be rejected")
	}
	catalog, _ = f.store.GameTitles(ctx, false)
	in.CatalogRevision = catalog.Revision
	in.AllowedIDs = nil
	state, err = f.store.ReconcileGameTitle(ctx, "Bob", in)
	if err != nil || state.TitleID != store.DefaultTitleID {
		t.Fatalf("all expired: %+v %v", state, err)
	}
	in.TitleID = b.ID
	in.ClientRequestID = "denied"
	in.ExpectedVersion = state.Version
	if _, err = f.store.ReconcileGameTitle(ctx, "Bob", in); !errors.Is(err, store.ErrUnauthorized) {
		t.Fatal("unauthorized title selected")
	}
	in.AllowedIDs = []string{b.ID}
	in.ExpectedVersion--
	if _, err = f.store.ReconcileGameTitle(ctx, "Bob", in); !errors.Is(err, store.ErrConflict) {
		t.Fatal("stale version accepted")
	}
	// Two devices selecting against the same version cannot silently overwrite each other.
	in.ExpectedVersion = state.Version
	in.AllowedIDs = []string{a.ID, b.ID}
	results := make(chan error, 2)
	for i, id := range []string{a.ID, b.ID} {
		request := in
		request.TitleID = id
		request.ClientRequestID = []string{"race-a", "race-b"}[i]
		go func() { _, e := f.store.ReconcileGameTitle(ctx, "Bob", request); results <- e }()
	}
	accepted, conflicts := 0, 0
	for range 2 {
		e := <-results
		if e == nil {
			accepted++
		} else if errors.Is(e, store.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(e)
		}
	}
	if accepted != 1 || conflicts != 1 {
		t.Fatalf("concurrent choices: accepted=%d conflicts=%d", accepted, conflicts)
	}
	rename.ExpectedVersion = a.Version
	rename.Status = "DELETED"
	rename.ClientRequestID = "delete-a"
	if _, err = f.store.ChangeGameTitle(ctx, f.users["Alice"].ID, a.ID, rename); err != nil {
		t.Fatal(err)
	}
	rename.ExpectedVersion = 0
	rename.Status = "PUBLISHED"
	rename.ClientRequestID = "reuse-a"
	if _, err = f.store.ChangeGameTitle(ctx, f.users["Alice"].ID, "", rename); err == nil {
		t.Fatal("deleted internal ID reused")
	}
	f.request(t, "Alice", "POST", "/api/v1/admin/titles/title_default", store.GameTitleChange{ClientRequestID: "delete-default", ExpectedVersion: 1, InternalID: "default", DisplayName: "溯行者", Status: "DELETED"}, 400)
}

func TestGameTitleIntroductionAndVisibilityAreVersionedMetadata(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	if _, err := f.store.DB.Exec("INSERT INTO identity_permissions VALUES(?,'platform.admin')", f.users["Alice"].ID); err != nil {
		t.Fatal(err)
	}
	in := store.GameTitleChange{ClientRequestID: "intro-create", InternalID: "private_title", DisplayName: "定制铭牌", Description: "介绍：星海同行。\n获取方式：完成纪念任务。", Public: false, Status: "PUBLISHED"}
	data := socialDataV2(f.request(t, "Alice", "POST", "/api/v1/admin/titles", in, 200))
	id := data["id"].(string)
	if data["description"] != in.Description || data["public"] != false {
		t.Fatalf("metadata lost: %v", data)
	}
	in.ClientRequestID = "intro-public"
	in.ExpectedVersion = 1
	in.Public = true
	in.Description = "公开活动获取。"
	changed := socialDataV2(f.request(t, "Alice", "POST", "/api/v1/admin/titles/"+id, in, 200))
	if changed["id"] != id || changed["public"] != true || changed["publishSequence"] != data["publishSequence"] {
		t.Fatalf("metadata changed identity/order: %v", changed)
	}
	catalog, err := f.store.GameTitles(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, title := range catalog.Items {
		if title.ID == id {
			found = title.Public && title.Description == in.Description
		}
	}
	if !found {
		t.Fatal("Core catalog lacks updated introduction/visibility")
	}
}
