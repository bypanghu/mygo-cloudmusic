package netease

import (
	"fmt"
	"net/url"
	"strconv"
)

// Podcast is a channel, not a song; its ID must never enter the song queue.
type Podcast struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	PicURL       string `json:"picUrl"`
	Description  string `json:"desc"`
	Category     string `json:"category"`
	DJ           User   `json:"dj"`
	ProgramCount int    `json:"programCount"`
	SubCount     int64  `json:"subCount"`
	Subed        bool   `json:"subed"`
}

type PodcastProgram struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	CoverURL    string  `json:"coverUrl"`
	Duration    int64   `json:"duration"`
	CreateTime  int64   `json:"createTime"`
	Description string  `json:"description"`
	MainSong    Track   `json:"mainSong"`
	Radio       Podcast `json:"radio"`
	DJ          User    `json:"dj"`
}

type PodcastPrograms struct {
	Programs []PodcastProgram `json:"programs"`
	Count    int              `json:"count"`
	More     bool             `json:"more"`
}

func PodcastDetail(id int64) (*Podcast, error) {
	var res struct {
		Data *Podcast `json:"data"`
	}
	if err := get("/api/djradio/v2/get", url.Values{"id": {strconv.FormatInt(id, 10)}}, &res); err != nil {
		return nil, err
	}
	if res.Data == nil || res.Data.ID <= 0 || res.Data.ID != id {
		return nil, fmt.Errorf("netease: podcast %d not found", id)
	}
	return res.Data, nil
}

// PodcastProgramDetail resolves the underlying audio when a list omits mainSong.
func PodcastProgramDetail(id int64) (*PodcastProgram, error) {
	var res struct {
		Program PodcastProgram `json:"program"`
	}
	if err := get("/api/dj/program/detail", url.Values{"id": {strconv.FormatInt(id, 10)}}, &res); err != nil {
		return nil, err
	}
	return &res.Program, nil
}

// Track uses mainSong's audio ID, while keeping program and channel IDs for
// navigation and session restoration; these IDs are not interchangeable.
func (p *PodcastProgram) Track(channel *Podcast) (Track, bool) {
	if p.ID <= 0 || p.MainSong.ID <= 0 {
		return Track{}, false
	}
	radio := p.Radio
	if channel != nil {
		radio = *channel
	}
	t := p.MainSong
	t.ProgramID, t.PodcastID = p.ID, radio.ID
	t.Description = p.Description
	if p.Name != "" {
		t.Name = p.Name
	}
	if p.Duration > 0 {
		t.Duration = p.Duration
	}
	cover := p.CoverURL
	if cover == "" {
		cover = radio.PicURL
	}
	if cover == "" {
		cover = t.Album.PicURL
	}
	t.Album = Album{Name: radio.Name, PicURL: cover}
	dj := p.DJ
	if dj.Nickname == "" {
		dj = radio.DJ
	}
	t.Artists = nil
	if dj.Nickname != "" {
		t.Artists = []Artist{{Name: dj.Nickname}}
	}
	return t, true
}

func PodcastEpisodes(id int64, limit, offset int) (PodcastPrograms, error) {
	var res PodcastPrograms
	err := get("/api/dj/program/byradio", url.Values{
		"radioId": {strconv.FormatInt(id, 10)},
		"limit":   {strconv.Itoa(limit)},
		"offset":  {strconv.Itoa(offset)},
		"asc":     {"false"},
	}, &res)
	return res, err
}
