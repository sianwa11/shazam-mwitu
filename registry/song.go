package registry

type SongInfo struct {
	Name string
	ID   int
	Path string
}

type SongRegistry struct {
	songs  map[int]SongInfo
	nextID int
}

func NewSongRegistry() *SongRegistry {
	return &SongRegistry{
		songs:  make(map[int]SongInfo),
		nextID: 1,
	}
}

func (r *SongRegistry) Register(title, path string) SongInfo {
	info := SongInfo{
		Name: title,
		Path: path,
		ID:   r.nextID,
	}

	r.songs[info.ID] = info
	r.nextID++
	return info
}
