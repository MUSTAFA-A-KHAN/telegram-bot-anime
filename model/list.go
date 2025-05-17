package model

type List struct {
	ID    int
	Name  string
	Count int
}
type Reaction struct {
	Type  string `json:"type"`
	Emoji string `json:"emoji"`
}
type ReactionRequest struct {
	ChatID    int64      `json:"chat_id"`
	MessageID int        `json:"message_id"`
	Reaction  []Reaction `json:"reaction"`
	IsBig     bool       `json:"is_big"`
}
