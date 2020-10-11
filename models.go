package main

type Event struct {
	CreatedAt string `json:"created_at"`
	ID        int    `json:"id"`
	Type      string `json:"type"`

	// type: achievement
	Achievement Achievement `json:"achievement,omitempty"`

	// type: beatmapsetApprove
	// type: beatmapsetDelete
	// type: beatmapsetRevive
	// type: beatmapsetUpdate
	// type: beatmapsetUpload
	Beatmapset Beatmapset `json:"beatmapset,omitempty"`

	User User `json:"user,omitempty"`
}

type Achievement struct{}

type Beatmapset struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type User struct {
	Username         string `json:"username"`
	URL              string `json:"url"`
	PreviousUsername string `json:"previousUsername,omitempty"`
}
