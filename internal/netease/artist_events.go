package netease

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type ArtistEventPage struct {
	Events   []ArtistEvent
	LastTime int64
	More     bool
	UserID   int64
}

type ArtistEvent struct {
	ID       int64                `json:"id"`
	Type     int                  `json:"type"`
	Time     int64                `json:"eventTime"`
	User     User                 `json:"user"`
	Pictures []ArtistEventPicture `json:"pics"`
	Info     ArtistEventInfo      `json:"info"`
	Content  ArtistEventContent   `json:"-"`
}

type ArtistEventPicture struct {
	OriginURL   string `json:"originUrl"`
	SquareURL   string `json:"squareUrl"`
	PCSquareURL string `json:"pcSquareUrl"`
}

type ArtistEventInfo struct {
	CommentCount int `json:"commentCount"`
	LikedCount   int `json:"likedCount"`
	ShareCount   int `json:"shareCount"`
}

type ArtistEventContent struct {
	Msg           string
	Song          *Track
	Album         *Album
	Artist        *Artist
	Playlist      *Playlist
	Video         json.RawMessage
	MV            json.RawMessage
	ForwardedText string
}

// ArtistEvents resolves the artist's bound account before reading its public feed.
// Artist IDs and account IDs belong to different namespaces.
func ArtistEvents(artistID int64, limit int, lasttime int64) (ArtistEventPage, error) {
	if artistID <= 0 {
		return ArtistEventPage{}, fmt.Errorf("netease: invalid artist ID")
	}
	uid, err := artistEventUserID(artistID)
	if err != nil {
		return ArtistEventPage{}, err
	}
	if uid <= 0 {
		return ArtistEventPage{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	if lasttime == 0 {
		lasttime = -1
	}
	var res struct {
		Events   []json.RawMessage `json:"events"`
		LastTime int64             `json:"lasttime"`
		More     bool              `json:"more"`
	}
	q := url.Values{"getcounts": {"true"}, "time": {strconv.FormatInt(lasttime, 10)}, "limit": {strconv.Itoa(limit)}, "total": {"false"}, "fromRN": {"true"}}
	if err := get("/api/event/get/"+strconv.FormatInt(uid, 10), q, &res); err != nil {
		return ArtistEventPage{}, err
	}
	page := ArtistEventPage{LastTime: res.LastTime, More: res.More, UserID: uid}
	for _, raw := range res.Events {
		var event ArtistEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			return ArtistEventPage{}, err
		}
		var payload struct {
			JSON json.RawMessage `json:"json"`
		}
		// The outer event remains useful even when one shared object is malformed.
		_ = json.Unmarshal(raw, &payload)
		event.Content = decodeArtistEventContent(payload.JSON, true)
		page.Events = append(page.Events, event)
	}
	return page, nil
}

func artistEventUserID(id int64) (int64, error) {
	type binding struct {
		UserID int64 `json:"userId"`
		UID    int64 `json:"uid"`
	}
	var head struct {
		Data struct {
			User   binding `json:"user"`
			Artist binding `json:"artist"`
		} `json:"data"`
	}
	headErr := get("/api/artist/head/info/get", url.Values{"id": {strconv.FormatInt(id, 10)}}, &head)
	for _, uid := range []int64{head.Data.User.UserID, head.Data.User.UID, head.Data.Artist.UserID} {
		if headErr == nil && uid > 0 {
			return uid, nil
		}
	}
	var legacy struct {
		UserID int64   `json:"userId"`
		Artist binding `json:"artist"`
	}
	err := get("/api/artist/"+strconv.FormatInt(id, 10), nil, &legacy)
	if err != nil {
		if headErr == nil {
			return 0, nil
		}
		return 0, err
	}
	if legacy.UserID > 0 {
		return legacy.UserID, nil
	}
	return legacy.Artist.UserID, nil
}

// Decode fields independently so a bad attachment cannot discard the message.
// The service normally wraps this object in a JSON string; accept an object too.
func decodeArtistEventContent(raw json.RawMessage, forwarded bool) ArtistEventContent {
	var out ArtistEventContent
	if len(raw) == 0 || string(raw) == "null" {
		return out
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		raw = json.RawMessage(text)
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		out.Msg = strings.TrimSpace(string(raw))
		return out
	}
	_ = json.Unmarshal(fields["msg"], &out.Msg)
	if value := fields["song"]; len(value) > 0 {
		var song Track
		if json.Unmarshal(value, &song) == nil && song.ID > 0 {
			out.Song = &song
		}
	}
	if value := fields["album"]; len(value) > 0 {
		var album Album
		if json.Unmarshal(value, &album) == nil && album.ID > 0 {
			out.Album = &album
		}
	}
	if value := fields["artist"]; len(value) > 0 {
		var artist Artist
		if json.Unmarshal(value, &artist) == nil && artist.ID > 0 {
			out.Artist = &artist
		}
	}
	if value := fields["playlist"]; len(value) > 0 {
		var playlist Playlist
		if json.Unmarshal(value, &playlist) == nil && playlist.ID > 0 {
			out.Playlist = &playlist
		}
	}
	out.Video, out.MV = fields["video"], fields["mv"]
	if forwarded && len(fields["event"]) > 0 {
		var original struct {
			User User            `json:"user"`
			JSON json.RawMessage `json:"json"`
		}
		if json.Unmarshal(fields["event"], &original) == nil {
			content := decodeArtistEventContent(original.JSON, false)
			if content.Msg != "" {
				out.ForwardedText = content.Msg
				if original.User.Nickname != "" {
					out.ForwardedText = original.User.Nickname + "：" + content.Msg
				}
			}
		}
	}
	return out
}
