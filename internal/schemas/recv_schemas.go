package schemas

import (
	"context"
	"fmt"
	"github.com/cohesivestack/valgo"
	"github.com/google/uuid"
	"party-buddy/internal/configuration"
	"party-buddy/internal/schemas/api"
	"party-buddy/internal/validate"
	"regexp"
)

var (
	baseReg        = regexp.MustCompile(fmt.Sprintf("[%v]", configuration.BaseTextFieldTemplate))
	checkedTextReg = regexp.MustCompile(fmt.Sprintf("[%v]", configuration.CheckedTextAnswerTemplate))
)

type GameType string

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
			InSlice([]GameType{Public, Private}, "game-type"))
}

type PublicCreateSessionRequest struct {
	BaseCreateSessionRequest
	GameID *uuid.UUID `json:"game-id"`
}

func (r *PublicCreateSessionRequest) Validate(ctx context.Context) *valgo.Validation {
	return r.BaseCreateSessionRequest.Validate(ctx).
		Is(valgo.StringP(r.GameType, "game-type", "game-type").EqualTo(Public))
}

type PrivateCreateSessionRequest struct {
	BaseCreateSessionRequest
	Game *FullGameInfo `json:"game"`
}

func (r *PrivateCreateSessionRequest) Validate(ctx context.Context) *valgo.Validation {
	return r.BaseCreateSessionRequest.Validate(ctx).
		Is(valgo.StringP(r.GameType, "game-type", "game-type").EqualTo(Private)).
		Merge(r.Game.Validate(ctx))
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
		MatchingTo(baseReg).MaxLength(configuration.MaxNameLength))
	v = v.Is(valgo.StringP(info.Description, "description", "description").Not().Nil().
		MatchingTo(baseReg).MaxLength(configuration.MaxDescriptionLength))
	v = v.Is(validate.FieldValue(info.Tasks).Set()).
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

	//AnswerIdx from ChoiceTask
	AnswerIndex *uint8 `json:"answer-idx,omitempty"`
}

func (t *BaseTaskWithImgRequest) Validate(ctx context.Context) *valgo.Validation {
	f, _ := validate.FromContext(ctx)

	v := f.Is(valgo.StringP(t.Name, "name", "name").Not().Nil().
		MatchingTo(baseReg).MaxLength(configuration.MaxNameLength))
	v = v.Is(valgo.StringP(t.Description, "description", "description").Not().Nil().
		MatchingTo(baseReg).MaxLength(configuration.MaxDescriptionLength))
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
		Is(valgo.StringP(t.Type, "type", "type").Not().Not().
			InSlice([]TaskType{Photo, Text, Choice, CheckedText}))

	if t.Type == nil {
		return v
	}

	switch *t.Type {
	case Photo:
		v = v.Is(valgo.StringP(t.Type, "type", "type").EqualTo(Photo)).
			Is(validate.FieldValue(t.PollDuration).Set()).
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
			Is(validate.FieldValue(t.PollDuration).Set()).
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
			Is(valgo.Uint8P(t.AnswerIndex, "answer", "answer").Not().Nil().
				LessThan(configuration.OptionsCount)).
			Is(validate.FieldValue(t.Options).Set()).
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
				MatchingTo(baseReg).MaxLength(configuration.MaxOptionLength))
		}
		return v

	case CheckedText:
		v = v.Is(valgo.StringP(t.Type, "type", "type").EqualTo(CheckedText)).
			Is(valgo.StringP(t.Answer, "answer", "answer").Not().Nil().
				MatchingTo(checkedTextReg).MaxLength(configuration.MaxCheckedTextAnswerLength))
		return v

	default:
		return v
	}
}
