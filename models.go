package main

type User struct {
	Id          int    `json:"id"`
	Username    string `json:"username"`
	CountryCode string `json:"country_code"`
}

type Beatmapset struct {
	Id int `json:"id"`

	Artist        string `json:"artist"`
	ArtistUnicode string `json:"artist_unicode"`
	Title         string `json:"title"`
	TitleUnicode  string `json:"title_unicode"`
	Creator       string `json:"creator"`
	UserId        int    `json:"user_id"`

	Covers   BeatmapCovers `json:"covers"`
	Beatmaps []Beatmap     `json:"beatmaps,omitempty"`
}

type Beatmap struct {
	Id               int     `json:"id"`
	DifficultyRating float64 `json:"difficulty_rating"`
	DifficultyName   string  `json:"version"`
}

type BeatmapCovers struct {
	Cover       string `json:"cover"`
	Cover2x     string `json:"cover@2x"`
	Card        string `json:"card"`
	Card2x      string `json:"card@2x"`
	SlimCover   string `json:"slimcover"`
	SlimCover2x string `json:"slimcover@2x"`
}

type Event struct {
	CreatedAt string `json:"created_at"`
	ID        int    `json:"id"`
	Type      string `json:"type"`

	// type: achievement
	Achievement EventAchievement `json:"achievement,omitempty"`

	// type: beatmapsetApprove
	// type: beatmapsetDelete
	// type: beatmapsetRevive
	// type: beatmapsetUpdate
	// type: beatmapsetUpload
	Beatmapset EventBeatmapset `json:"beatmapset,omitempty"`

	User EventUser `json:"user,omitempty"`
}

type EventAchievement struct{}

type EventBeatmapset struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type EventUser struct {
	Username         string `json:"username"`
	URL              string `json:"url"`
	PreviousUsername string `json:"previousUsername,omitempty"`
}
