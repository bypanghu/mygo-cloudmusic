package app

import "netease-music-mygo/internal/netease"

var (
	fetchPodcastDetail   = netease.PodcastDetail
	fetchPodcastEpisodes = netease.PodcastEpisodes
	fetchPodcastProgram  = netease.PodcastProgramDetail
)

const podcastEpisodeLimit = 30

type catalogPodcast struct {
	data                                  *netease.Podcast
	loading                               bool
	err                                   string
	programs                              []netease.PodcastProgram
	programsLoaded, programsLoading, more bool
	programsErr                           string
	offset                                int
	resolving                             map[int64]bool
	playErr                               string
	subscribing                           bool
}

// playPodcastProgram queues the loaded episodes by their audio IDs, keeping
// the selected episode's index after entries without audio have been omitted.
func (a *app) playPodcastProgram(id, programID int64) {
	s := a.catalog.podcasts[id]
	if s == nil || s.data == nil {
		return
	}
	s.playErr = ""
	for i := range s.programs {
		if s.programs[i].ID != programID {
			continue
		}
		if _, ok := s.programs[i].Track(s.data); ok {
			queue := make([]netease.Track, 0, len(s.programs))
			start := -1
			for j := range s.programs {
				if track, ok := s.programs[j].Track(s.data); ok {
					if s.programs[j].ID == programID {
						start = len(queue)
					}
					queue = append(queue, track)
				}
			}
			if start >= 0 {
				a.catalog.programSerial++
				a.pl.PlayQueue(queue, start)
			}
			return
		}
		if s.resolving[programID] {
			return
		}
		if s.resolving == nil {
			s.resolving = make(map[int64]bool)
		}
		s.resolving[programID] = true
		a.catalog.programSerial++
		request, fetch := a.catalog.programSerial, fetchPodcastProgram
		current := a.pl.Current()
		go func() {
			program, err := fetch(programID)
			a.post(func() {
				delete(s.resolving, programID)
				if err != nil || program == nil || program.ID != programID || program.MainSong.ID <= 0 {
					s.playErr = "节目音频加载失败，请点击节目重试"
					return
				}
				for j := range s.programs {
					if s.programs[j].ID == programID {
						s.programs[j] = *program
						break
					}
				}
				// A newer choice, a different page, or changed playback wins
				// over a late response from this episode.
				if a.catalog.podcasts[id] == s && request == a.catalog.programSerial && a.route == (route{page: pagePodcast, entity: id}) && a.pl.Current() == current {
					a.playPodcastProgram(id, programID)
				}
			})
		}()
		return
	}
}

func (a *app) loadPodcast(id int64) {
	if a.catalog.podcasts == nil {
		a.catalog.podcasts = make(map[int64]*catalogPodcast)
	}
	s := a.catalog.podcasts[id]
	if s == nil {
		s = &catalogPodcast{}
		a.catalog.podcasts[id] = s
	}
	if !s.programsLoaded {
		a.loadPodcastEpisodes(id)
	}
	if s.data != nil || s.loading {
		return
	}
	s.loading, s.err = true, ""
	fetch := fetchPodcastDetail
	go func() {
		data, err := fetch(id)
		a.post(func() {
			s.loading = false
			if err != nil || data == nil {
				s.err = "播客详情加载失败"
				return
			}
			s.data = data
		})
	}()
}

func (a *app) loadPodcastEpisodes(id int64) {
	s := a.catalog.podcasts[id]
	if s == nil || s.programsLoading || (s.programsLoaded && !s.more) {
		return
	}
	s.programsLoading, s.programsErr = true, ""
	offset, fetch := s.offset, fetchPodcastEpisodes
	go func() {
		page, err := fetch(id, podcastEpisodeLimit, offset)
		a.post(func() {
			s.programsLoading = false
			if err != nil {
				s.programsErr = "节目列表加载失败"
				return
			}
			s.programsLoaded, s.more = true, page.More && len(page.Programs) > 0
			s.offset += len(page.Programs)
			seen := make(map[int64]bool, len(s.programs))
			for _, program := range s.programs {
				seen[program.ID] = true
			}
			for _, program := range page.Programs {
				if !seen[program.ID] {
					s.programs = append(s.programs, program)
					seen[program.ID] = true
				}
			}
		})
	}()
}
