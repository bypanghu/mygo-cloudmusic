package netease

import (
	"fmt"
	"strconv"
)

// SubscribePodcast subscribes to a channel, or removes its subscription.
func SubscribePodcast(id int64, subscribe bool) error {
	return setSubscription("djradio", subscribe, map[string]any{"id": strconv.FormatInt(id, 10)})
}

// FollowArtist saves an artist, not the artist's separate user account.
func FollowArtist(id int64, follow bool) error {
	return setSubscription("artist", follow, map[string]any{
		"artistId":  strconv.FormatInt(id, 10),
		"artistIds": fmt.Sprintf("[%d]", id),
	})
}

func setSubscription(kind string, subscribe bool, data map[string]any) error {
	op := "unsub"
	if subscribe {
		op = "sub"
	}
	path := "/api/" + kind + "/" + op
	var res struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := eapiPost(path, data, &res); err != nil {
		return err
	}
	if res.Code != 200 {
		return fmt.Errorf("netease: %s: code %d %s", path, res.Code, res.Message)
	}
	return nil
}
