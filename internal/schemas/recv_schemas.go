package schemas

import "github.com/google/uuid"

type ImgRequest int8

type GameType string

const (
	Public  GameType = "public"
	Private GameType = "private"
)

type BaseCreateSessionRequest struct {
	PlayerCount  int8     `json:"player-count"`
	RequireReady bool     `json:"require-ready"`
	GameType     GameType `json:"game-type"`
}

type PublicCreateSessionRequest struct {
	BaseCreateSessionRequest
	GameID uuid.UUID `json:"game-id"`
}

type PrivateCreateSessionRequest struct {
	BaseCreateSessionRequest
	Game FullGameInfo `json:"game"`
}

type FullGameInfo struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ImgRequest  ImgRequest `json:"img-request"`
}

type BaseTaskWithImgRequest struct {
	BaseTask
	ImgRequest ImgRequest `json:"img-request"`
}

type AnsweredCheckedTextTaskImgRequest struct {
	BaseTaskWithImgRequest

	Answer string `json:"answer"`
}

type AnsweredChoiceTaskImgRequest struct {
	BaseTaskWithImgRequest

	Options     []string `json:"options"`
	AnswerIndex uint8    `json:"answer-idx"`
}
