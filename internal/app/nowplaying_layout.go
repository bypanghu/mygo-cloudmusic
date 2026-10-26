package app

// npLayout keeps artwork and reading widths bounded on large screens while
// leaving room for the controls in the shortest supported windows.
type npLayout struct {
	width, side, cover float32
	gap, top, bottom   float32
}

func nowPlayingLayout(w, h, bar float32) npLayout {
	margin := min(72, max(32, w*0.05))
	width := min(1240, w-2*margin)
	gap := min(88, max(36, width*0.065))
	side := min(440, (width-gap)*0.43)
	top := min(40, max(12, h*0.035))
	bottom := min(48, max(20, h*0.045))
	cover := min(side, max(120, h-bar-top-bottom-286))
	return npLayout{width: width, side: side, cover: cover, gap: gap, top: top, bottom: bottom}
}
