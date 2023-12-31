package schemas

import (
	"context"
	"github.com/cohesivestack/valgo"
	"github.com/google/uuid"
	"party-buddy/internal/configuration"
	"party-buddy/internal/schemas/api"
	"party-buddy/internal/util"
	"party-buddy/internal/validate"
)

type GameType string

var validGameTypes = []GameType{Public, Private}

const (
	Public  GameType = "public"
	Private GameType = "private"
)

type BaseCreateSessionRequest struct {
	PlayerCount  *int8     `json:"player-count"`
	RequireReady *bool     `json:"require-ready"`
	GameType     *GameType `json:"game-type"`
}

func (r *BaseCreateSessionRequest) Validate(ctx context.Context) *valgo.Validation {
	f, _ := validate.FromContext(ctx)

	return f.
		Is(valgo.Int8P(r.PlayerCount, "player-count", "player-count").Not().Nil().
			Between(configuration.PlayerMin, configuration.PlayerMax)).
		Is(valgo.StringP(r.GameType, "game-type", "game-type").Not().Nil().
			InSlice(validGameTypes, "game-type")).
		Is(valgo.BoolP(r.RequireReady, "require-ready", "require-ready").Not().Nil())
}

type PublicCreateSessionRequest struct {
	BaseCreateSessionRequest
	GameID *uuid.UUID `json:"game-id"`
}

func (r *PublicCreateSessionRequest) Validate(ctx context.Context) *valgo.Validation {
	return r.BaseCreateSessionRequest.Validate(ctx).
		Is(valgo.StringP(r.GameType, "game-type", "game-type").EqualTo(Public)).
		Is(validate.FieldValue(r.GameID, "game-id", "game-id").Set())
}

type PrivateCreateSessionRequest struct {
	BaseCreateSessionRequest
	Game *FullGameInfo `json:"game"`
}

func (r *PrivateCreateSessionRequest) Validate(ctx context.Context) *valgo.Validation {
	v := r.BaseCreateSessionRequest.Validate(ctx).
		Is(valgo.StringP(r.GameType, "game-type", "game-type").EqualTo(Private)).
		Is(validate.FieldValue(r.Game, "game", "game").Set())
	if r.Game == nil {
		return v
	}
	return v.Merge(r.Game.Validate(ctx))
}

type FullGameInfo struct {
	Name        *string                   `json:"name"`
	Description *string                   `json:"description"`
	ImgRequest  *api.ImgRequest           `json:"img-request"`
	Tasks       *[]BaseTaskWithImgRequest `json:"tasks"`
}

func (info *FullGameInfo) Validate(ctx context.Context) *valgo.Validation {
	f, _ := validate.FromContext(ctx)

	v := f.Is(valgo.StringP(info.Name, "name", "name").Not().Nil().
		MatchingTo(configuration.BaseTextReg).Passing(util.MaxLengthPChecker(configuration.MaxNameLength)))
	v = v.Is(valgo.StringP(info.Description, "description", "description").Not().Nil().
		MatchingTo(configuration.BaseTextReg).Passing(util.MaxLengthPChecker(configuration.MaxDescriptionLength)))
	v = v.Is(valgo.Int8P(info.ImgRequest, "img-request", "img-request").Not().Nil())
	v = v.Is(validate.FieldValue(info.Tasks, "tasks", "tasks").Set()).
		Is(valgo.Any(info.Tasks).Passing(func(v any) bool {
			tasks := v.(*[]BaseTaskWithImgRequest)
			if tasks == nil {
				return false
			}
			return len(*tasks) >= configuration.MinTaskCount && len(*tasks) <= configuration.MaxTaskCount
		}))
	if info.Tasks == nil {
		return v
	}
	for i := 0; i < len(*info.Tasks); i++ {
		v = v.Merge((*info.Tasks)[i].Validate(ctx))
	}
	return v
}

type BaseTaskWithImgRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`

	// Duration must be Fixed
	Duration *PollDuration `json:"duration"`

	Type *TaskType `json:"type"`

	PollDuration *PollDuration `json:"poll-duration,omitempty"`

	ImgRequest *api.ImgRequest `json:"img-request"`

	// Answer from CheckedTextTask
	Answer *string `json:"answer,omitempty"`

	// Options from ChoiceTask
	Options *[]string `json:"options,omitempty"`

	// AnswerIndex from ChoiceTask
	AnswerIndex *uint8 `json:"answer-idx,omitempty"`
}

func (t *BaseTaskWithImgRequest) Validate(ctx context.Context) *valgo.Validation {
	f, _ := validate.FromContext(ctx)

	v := f.Is(valgo.StringP(t.Name, "name", "name").Not().Nil().
		MatchingTo(configuration.BaseTextReg).Passing(util.MaxLengthPChecker(configuration.MaxNameLength)))
	v = v.Is(valgo.StringP(t.Description, "description", "description").Not().Nil().
		MatchingTo(configuration.BaseTextReg).Passing(util.MaxLengthPChecker(configuration.MaxDescriptionLength)))
	v = v.Is(valgo.Int8P(t.ImgRequest, "img-request", "img-request").Not().Nil())
	v = v.
		Is(validate.FieldValue(t.Duration, "duration", "duration").Set()).
		Is(valgo.Any(t.Duration, "duration", "duration").Passing(
			func(d any) bool {
				dur := d.(*PollDuration)
				if dur == nil {
					return false
				}
				return dur.Kind == Fixed
			})).
		Is(valgo.StringP(t.Type, "type", "type").Not().Nil().
			InSlice(validTaskTypes))

	if t.Type == nil {
		return v
	}

	switch *t.Type {
	case Photo:
		v = v.Is(valgo.StringP(t.Type, "type", "type").EqualTo(Photo)).
			Is(validate.FieldValue(t.PollDuration, "poll-duration", "poll-duration").Set()).
			Is(valgo.Any(t.PollDuration, "poll-duration", "poll-duration").Passing(func(v any) bool {
				d := v.(*PollDuration)
				if d == nil {
					return false
				}
				return d.Kind == Fixed || d.Kind == Dynamic
			}))
		return v

	case Text:
		v = v.Is(valgo.StringP(t.Type, "type", "type").EqualTo(Text)).
			Is(validate.FieldValue(t.PollDuration, "poll-duration", "poll-duration").Set()).
			Is(valgo.Any(t.PollDuration, "poll-duration", "poll-duration").Passing(func(v any) bool {
				d := v.(*PollDuration)
				if d == nil {
					return false
				}
				return d.Kind == Fixed || d.Kind == Dynamic
			}))
		return v

	case Choice:
		v = v.Is(valgo.StringP(t.Type, "type", "type").EqualTo(Choice)).
			Is(valgo.Uint8P(t.AnswerIndex, "answer-idx", "answer-idx").Not().Nil().
				LessThan(configuration.OptionsCount)).
			Is(validate.FieldValue(t.Options, "options", "options").Set()).
			Is(valgo.Any(t.Options, "options", "options").Passing(func(v any) bool {
				opts := v.(*[]string)
				if opts == nil {
					return false
				}
				return len(*opts) == configuration.OptionsCount
			}))
		if t.Options == nil {
			return v
		}

		for i := 0; i < len(*t.Options); i++ {
			v = v.Is(valgo.String((*t.Options)[i], "option", "option").
				MatchingTo(configuration.BaseTextReg).Passing(util.MaxLengthChecker(configuration.MaxOptionLength)))
		}
		return v

	case CheckedText:
		v = v.Is(valgo.StringP(t.Type, "type", "type").EqualTo(CheckedText)).
			Is(valgo.StringP(t.Answer, "answer", "answer").Not().Nil().
				MatchingTo(configuration.CheckedTextAnswerReg).
				Passing(util.MaxLengthPChecker(configuration.MaxCheckedTextAnswerLength)))
		return v

	default:
		return v
	}
}
