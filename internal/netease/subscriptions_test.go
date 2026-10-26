package netease

import (
	"bytes"
	"crypto/aes"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// Podcast details live under data, not djRadio; reading the wrong envelope used to erase every count.
func TestPodcastDetailCounts(t *testing.T) {
	defer mockCatalog(func(r *http.Request) string {
		if r.URL.Path != "/api/djradio/v2/get" || r.URL.Query().Get("id") != "7" {
			t.Fatalf("wrong podcast request: %s", r.URL)
		}
		return `{"code":200,"data":{"id":7,"name":"channel","picUrl":"cover","desc":"description","programCount":403,"subCount":89986,"subed":true,"dj":{"userId":8,"nickname":"host"}}}`
	})()
	got, err := PodcastDetail(7)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 7 || got.Name != "channel" || got.PicURL != "cover" || got.Description != "description" || got.ProgramCount != 403 || got.SubCount != 89986 || !got.Subed || got.DJ.UserID != 8 {
		t.Fatalf("podcast metadata lost: %+v", got)
	}
}

// A missing, empty, or mismatched podcast must fail instead of rendering an empty detail page.
func TestPodcastDetailMissing(t *testing.T) {
	for _, body := range []string{
		`{"code":200}`,
		`{"code":200,"data":null}`,
		`{"code":200,"data":{}}`,
		`{"code":200,"data":{"id":8}}`,
		`{"code":404,"message":"not found"}`,
	} {
		t.Run(body, func(t *testing.T) {
			defer mockCatalog(func(*http.Request) string { return body })()
			if got, err := PodcastDetail(7); err == nil || got != nil {
				t.Fatalf("accepted missing detail: %+v / %v", got, err)
			}
		})
	}
}

// Artist details retain the server's follow state so a saved artist is not offered a second follow.
func TestArtistDetailFollowed(t *testing.T) {
	defer mockCatalog(func(r *http.Request) string {
		if r.URL.Path != "/api/artist/7" {
			t.Fatalf("unexpected request: %s", r.URL)
		}
		return `{"code":200,"artist":{"id":7,"name":"artist","followed":true},"hotSongs":[]}`
	})()
	got, err := ArtistDetail(7)
	if err != nil || got == nil || !got.Artist.Followed {
		t.Fatalf("follow state lost: %+v / %v", got, err)
	}
}

// Subscribing and following send their distinct entity IDs to the matching authenticated endpoints.
func TestSubscriptionRequests(t *testing.T) {
	deviceOnce.Do(func() { deviceID = "0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123" })
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	for _, tt := range []struct {
		name string
		path string
		data map[string]any
		send func() error
	}{
		{"subscribe podcast", "/api/djradio/sub", map[string]any{"id": "7"}, func() error { return SubscribePodcast(7, true) }},
		{"unsubscribe podcast", "/api/djradio/unsub", map[string]any{"id": "7"}, func() error { return SubscribePodcast(7, false) }},
		{"follow artist", "/api/artist/sub", map[string]any{"artistId": "9", "artistIds": "[9]"}, func() error { return FollowArtist(9, true) }},
		{"unfollow artist", "/api/artist/unsub", map[string]any{"artistId": "9", "artistIds": "[9]"}, func() error { return FollowArtist(9, false) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			http.DefaultTransport = catalogTransport(func(r *http.Request) string {
				called = true
				if r.Method != http.MethodPost || r.URL.Path != strings.Replace(tt.path, "/api/", "/eapi/", 1) {
					t.Fatalf("wrong mutation endpoint: %s %s", r.Method, r.URL)
				}
				path, data := subscriptionPayload(t, r)
				delete(data, "header")
				delete(data, "e_r")
				if path != tt.path || !reflect.DeepEqual(data, tt.data) {
					t.Fatalf("wrong subscription payload: %s / %+v", path, data)
				}
				return `{"code":200}`
			})
			if err := tt.send(); err != nil || !called {
				t.Fatalf("subscription request: called %v / %v", called, err)
			}
		})
	}
}

// A rejected or malformed subscription response must not be reported as a successful mutation.
func TestSubscriptionRejections(t *testing.T) {
	deviceOnce.Do(func() { deviceID = "0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123" })
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	for _, body := range []string{`{"code":301}`, `{"code":500,"message":"unavailable"}`, `{}`, `invalid`} {
		t.Run(body, func(t *testing.T) {
			http.DefaultTransport = catalogTransport(func(*http.Request) string { return body })
			if err := SubscribePodcast(7, true); err == nil {
				t.Fatal("podcast rejection lost")
			}
			if err := FollowArtist(9, false); err == nil {
				t.Fatal("artist rejection lost")
			}
		})
	}
}

func subscriptionPayload(t *testing.T, r *http.Request) (string, map[string]any) {
	t.Helper()
	if err := r.ParseForm(); err != nil {
		t.Fatal(err)
	}
	sealed, err := hex.DecodeString(r.Form.Get("params"))
	if err != nil || len(sealed) == 0 || len(sealed)%aes.BlockSize != 0 {
		t.Fatalf("invalid sealed request: %v", err)
	}
	block, err := aes.NewCipher([]byte(eapiKey))
	if err != nil {
		t.Fatal(err)
	}
	plain := make([]byte, len(sealed))
	for i := 0; i < len(sealed); i += aes.BlockSize {
		block.Decrypt(plain[i:i+aes.BlockSize], sealed[i:i+aes.BlockSize])
	}
	parts := bytes.Split(plain, []byte("-36cd479b6b5-"))
	if len(parts) != 3 {
		t.Fatal("invalid eapi envelope")
	}
	var data map[string]any
	if err := json.Unmarshal(parts[1], &data); err != nil {
		t.Fatal(err)
	}
	return string(parts[0]), data
}
