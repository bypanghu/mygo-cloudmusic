package app

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

// Podcast subscription clicks send the channel ID, adjust its count once, and undo a refused change.
func TestPodcastSubscriptionButton(t *testing.T) {
	old := subscribePodcast
	defer func() { subscribePodcast = old }()
	for _, subscribed := range []bool{false, true} {
		name, action, nextAction := "subscribe", "订阅播客", "取消订阅播客"
		if subscribed {
			name, action, nextAction = "unsubscribe", nextAction, action
		}
		t.Run(name, func(t *testing.T) {
			type request struct {
				id        int64
				subscribe bool
			}
			calls := make(chan request, 2)
			subscribePodcast = func(id int64, subscribe bool) error {
				calls <- request{id, subscribe}
				return nil
			}
			a := newTestApp()
			a.settings = defaultSettings()
			a.profile = &netease.Profile{UserID: 1}
			s := &catalogPodcast{
				data:           &netease.Podcast{ID: 7, Name: "在云端", ProgramCount: 403, SubCount: 42, Subed: subscribed},
				programsLoaded: true,
			}
			a.catalog.podcasts = map[int64]*catalogPodcast{7: s}
			a.open(route{page: pagePodcast, entity: 7})
			tt := ui.NewTester(a.view, 1000, 700)
			if err := tt.Click(action); err != nil {
				t.Fatal(err)
			}
			wantCount := int64(43)
			if subscribed {
				wantCount = 41
			}
			if !s.subscribing || s.data.Subed == subscribed || s.data.SubCount != wantCount {
				t.Fatalf("click did not update subscription: %+v", s)
			}
			if got := <-calls; got.id != 7 || got.subscribe == subscribed {
				t.Fatalf("wrong subscription request: %+v", got)
			}
			tt.Frame()
			if err := tt.Click(nextAction); err != nil {
				t.Fatal(err)
			}
			if s.data.Subed == subscribed || s.data.SubCount != wantCount {
				t.Fatal("a pending subscription was toggled twice")
			}
			a.completePodcastSubscription(7, s, a.profile, !subscribed, 42, errors.New("offline"))
			if s.subscribing || s.data.Subed != subscribed || s.data.SubCount != 42 || a.toast == "" {
				t.Fatalf("refused subscription did not roll back: %+v", s)
			}
			tt.Frame()
			if err := tt.Click(action); err != nil {
				t.Fatal(err)
			}
			if got := <-calls; got.id != 7 || got.subscribe == subscribed {
				t.Fatalf("retry sent a different operation: %+v", got)
			}
			a.completePodcastSubscription(7, s, a.profile, !subscribed, 42, nil)
			if s.subscribing || s.data.Subed == subscribed || s.data.SubCount != wantCount {
				t.Fatalf("successful retry lost its state: %+v", s)
			}
		})
	}
}

// Artist follow clicks use the artist ID, ignore a pending repeat, and restore the state after rejection.
func TestArtistFollowButton(t *testing.T) {
	old := followArtist
	defer func() { followArtist = old }()
	for _, followed := range []bool{false, true} {
		name, action, nextAction := "follow", "关注音乐人", "取消关注音乐人"
		if followed {
			name, action, nextAction = "unfollow", nextAction, action
		}
		t.Run(name, func(t *testing.T) {
			type request struct {
				id     int64
				follow bool
			}
			calls := make(chan request, 2)
			followArtist = func(id int64, follow bool) error {
				calls <- request{id, follow}
				return nil
			}
			a := newTestApp()
			a.settings = defaultSettings()
			a.profile = &netease.Profile{UserID: 1}
			s := &catalogArtist{data: &netease.ArtistPage{Artist: netease.Artist{ID: 7, Name: "MyGO!!!!!", Followed: followed}}}
			a.catalog.artists = map[int64]*catalogArtist{7: s}
			a.open(route{page: pageArtist, entity: 7})
			tt := ui.NewTester(a.view, 1000, 700)
			if err := tt.Click(action); err != nil {
				t.Fatal(err)
			}
			if !s.following || s.data.Artist.Followed == followed {
				t.Fatalf("click did not update follow state: %+v", s)
			}
			if got := <-calls; got.id != 7 || got.follow == followed {
				t.Fatalf("wrong follow request: %+v", got)
			}
			tt.Frame()
			if err := tt.Click(nextAction); err != nil {
				t.Fatal(err)
			}
			if s.data.Artist.Followed == followed {
				t.Fatal("a pending follow was toggled twice")
			}
			a.completeArtistFollow(7, s, a.profile, !followed, errors.New("offline"))
			if s.following || s.data.Artist.Followed != followed || a.toast == "" {
				t.Fatalf("refused follow did not roll back: %+v", s)
			}
			tt.Frame()
			if err := tt.Click(action); err != nil {
				t.Fatal(err)
			}
			if got := <-calls; got.id != 7 || got.follow == followed {
				t.Fatalf("retry sent a different operation: %+v", got)
			}
			a.completeArtistFollow(7, s, a.profile, !followed, nil)
			if s.following || s.data.Artist.Followed == followed {
				t.Fatalf("successful retry lost its state: %+v", s)
			}
		})
	}
}

// Signed-out subscription and follow clicks open login without changing state or sending a mutation.
func TestSubscriptionButtonsRequireLogin(t *testing.T) {
	oldSubscribe, oldFollow, oldTransport := subscribePodcast, followArtist, http.DefaultTransport
	defer func() {
		subscribePodcast, followArtist, http.DefaultTransport = oldSubscribe, oldFollow, oldTransport
	}()
	loginRequests := make(chan struct{}, 2)
	http.DefaultTransport = subscriptionLoginTransport{loginRequests}
	calls := make(chan struct{}, 2)
	subscribePodcast = func(int64, bool) error { calls <- struct{}{}; return nil }
	followArtist = func(int64, bool) error { calls <- struct{}{}; return nil }
	for _, page := range []pageKind{pagePodcast, pageArtist} {
		a := newTestApp()
		a.settings = defaultSettings()
		a.catalog.podcasts = map[int64]*catalogPodcast{7: {
			data: &netease.Podcast{ID: 7, Name: "在云端", SubCount: 42}, programsLoaded: true,
		}}
		a.catalog.artists = map[int64]*catalogArtist{7: {
			data: &netease.ArtistPage{Artist: netease.Artist{ID: 7, Name: "MyGO!!!!!"}},
		}}
		a.open(route{page: page, entity: 7})
		tt := ui.NewTester(a.view, 1000, 700)
		action := "订阅播客"
		if page == pageArtist {
			action = "关注音乐人"
		}
		if err := tt.Click(action); err != nil {
			t.Fatal(err)
		}
		if !a.login.open || a.catalog.podcasts[7].data.Subed || a.catalog.podcasts[7].data.SubCount != 42 || a.catalog.artists[7].data.Artist.Followed {
			t.Fatal("signed-out click did not preserve state and request login")
		}
		<-loginRequests
	}
	if len(calls) != 0 {
		t.Fatal("a signed-out click sent a subscription request")
	}
}

type subscriptionLoginTransport struct{ requests chan struct{} }

func (s subscriptionLoginTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	s.requests <- struct{}{}
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"code":500}`)), Request: r}, nil
}

// A late subscription response updates its own entity after navigation, but never a replacement account's state.
func TestSubscriptionCompletionScope(t *testing.T) {
	a := newTestApp()
	a.profile = &netease.Profile{UserID: 1}
	profile := a.profile
	podcast := &catalogPodcast{data: &netease.Podcast{ID: 7, Subed: true, SubCount: 43}, subscribing: true}
	artist := &catalogArtist{data: &netease.ArtistPage{Artist: netease.Artist{ID: 7, Followed: true}}, following: true}
	a.catalog.podcasts = map[int64]*catalogPodcast{7: podcast}
	a.catalog.artists = map[int64]*catalogArtist{7: artist}
	a.route = route{page: pageHome}
	a.completePodcastSubscription(7, podcast, profile, true, 42, errors.New("offline"))
	a.completeArtistFollow(7, artist, profile, true, errors.New("offline"))
	if podcast.subscribing || podcast.data.Subed || podcast.data.SubCount != 42 || artist.following || artist.data.Artist.Followed {
		t.Fatal("navigation discarded a completion for a cached entity")
	}
	// Signing out and back in can use the same user ID, but not the old cache.
	a.profile = &netease.Profile{UserID: 1}
	a.catalog.podcasts[7] = &catalogPodcast{data: &netease.Podcast{ID: 7, Subed: true, SubCount: 99}}
	a.catalog.artists[7] = &catalogArtist{data: &netease.ArtistPage{Artist: netease.Artist{ID: 7, Followed: true}}}
	a.toast = ""
	a.completePodcastSubscription(7, podcast, profile, true, 42, errors.New("old request"))
	a.completeArtistFollow(7, artist, profile, true, errors.New("old request"))
	if a.toast != "" || !a.catalog.podcasts[7].data.Subed || a.catalog.podcasts[7].data.SubCount != 99 || !a.catalog.artists[7].data.Artist.Followed {
		t.Fatal("an old session changed the current subscription state")
	}
}
