package schemas

import (
	"context"
	"github.com/cohesivestack/valgo"
	"github.com/google/uuid"
	"party-buddy/internal/configuration"
	"party-buddy/internal/validate"
)

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

func (r *BaseCreateSessionRequest) Validate(ctx context.Context) *valgo.Validation {
	f, _ := validate.FromContext(ctx)

	return f.
		Is(valgo.Int8(r.PlayerCount, "player-count", "player-count").
			Between(configuration.PlayerMin, configuration.PlayerMax))
}

type PublicCreateSessionRequest struct {
	BaseCreateSessionRequest
	GameID uuid.UUID `json:"game-id"`
}

func (r *PublicCreateSessionRequest) Validate(ctx context.Context) *valgo.Validation {
	return r.BaseCreateSessionRequest.Validate(ctx).
		Is(valgo.String(r.GameType, "game-type", "game-type").EqualTo(Public))
}

type PrivateCreateSessionRequest struct {
	BaseCreateSessionRequest
	Game FullGameInfo `json:"game"`
}

func (r *PrivateCreateSessionRequest) Validate(ctx context.Context) *valgo.Validation {
	return r.BaseCreateSessionRequest.Validate(ctx).
		Is(valgo.String(r.GameType, "game-type", "game-type").EqualTo(Private))
}

type FullGameInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	ImgRequest  ImgRequest     `json:"img-request"`
	Tasks       []AnsweredTask `json:"tasks"`
}

func (info *FullGameInfo) Validate(ctx context.Context) *valgo.Validation {
	f, _ := validate.FromContext(ctx)
	// TODO

	return f.Is(valgo.String(info.Name, "name", "name"))
}

type AnsweredTask interface {
	valgo.Validator
	isAnsweredTask()
}

type BaseTaskWithImgRequest struct {
	BaseTask
	ImgRequest ImgRequest `json:"img-request"`
}

type AnsweredCheckedTextTaskImgRequest struct {
	BaseTaskWithImgRequest

	Answer string `json:"answer"`
}

func (*AnsweredCheckedTextTaskImgRequest) isAnsweredTask() {}

type AnsweredChoiceTaskImgRequest struct {
	BaseTaskWithImgRequest

	Options     []string `json:"options"`
	AnswerIndex uint8    `json:"answer-idx"`
}

func (*AnsweredChoiceTaskImgRequest) isAnsweredTask() {}

type AnsweredPhotoTaskImgRequest struct {
	BaseTaskWithImgRequest
}

func (*AnsweredPhotoTaskImgRequest) isAnsweredTask() {}

type AnsweredTextTaskImgRequest struct {
	BaseTaskWithImgRequest
}

func (*AnsweredTextTaskImgRequest) isAnsweredTask() {}
