package schemas

import (
	"context"
	"fmt"
	"github.com/cohesivestack/valgo"
	"github.com/google/uuid"
	"party-buddy/internal/configuration"
	"party-buddy/internal/util"
	"party-buddy/internal/validate"
	"regexp"
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
			Between(configuration.PlayerMin, configuration.PlayerMax)).
		Is(valgo.String(r.GameType, "game-type", "game-type").
			Not().Blank().InSlice([]GameType{Public, Private}, "game-type"))
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
	reg := regexp.MustCompile(fmt.Sprintf("[%v]", configuration.BaseTextFieldTemplate))

	info.Name = util.ReplaceEwith2Dots(info.Name)
	v := f.Is(valgo.String(info.Name, "name", "name").
		MatchingTo(reg).MaxLength(configuration.MaxNameLength))
	info.Description = util.ReplaceEwith2Dots(info.Description)
	v = v.Is(valgo.String(info.Description, "description", "description").
		MatchingTo(reg).MaxLength(configuration.MaxDescriptionLength))
	v = v.Is(valgo.Any(info.Tasks).Passing(func(v any) bool {
		tasks := v.([]AnsweredTask)
		return len(tasks) >= configuration.MinTaskCount && len(tasks) <= configuration.MaxTaskCount
	}))
	for i := 0; i < len(info.Tasks); i++ {
		v = v.Merge(info.Tasks[i].Validate(ctx))
	}
	return v
}

type AnsweredTask interface {
	Validate(ctx context.Context) *valgo.Validation
	isAnsweredTask()
}

type BaseTaskWithImgRequest struct {
	BaseTask
	ImgRequest ImgRequest `json:"img-request"`
}

func (t *BaseTaskWithImgRequest) Validate(ctx context.Context) *valgo.Validation {
	f, _ := validate.FromContext(ctx)
	baseReg := regexp.MustCompile(fmt.Sprintf("[%v]", configuration.BaseTextFieldTemplate))

	t.Name = util.ReplaceEwith2Dots(t.Name)
	v := f.Is(valgo.String(t.Name, "name", "name").
		MatchingTo(baseReg).MaxLength(configuration.MaxNameLength))
	t.Description = util.ReplaceEwith2Dots(t.Description)
	v = v.Is(valgo.String(t.Description, "description", "description").
		MatchingTo(baseReg).MaxLength(configuration.MaxDescriptionLength))
	return v.Is(valgo.Any(t.Duration, "duration", "duration").Passing(func(d any) bool {
		return d.(PollDuration).Kind == Fixed
	}))
}

type AnsweredCheckedTextTaskImgRequest struct {
	BaseTaskWithImgRequest

	Answer string `json:"answer"`
}

func (*AnsweredCheckedTextTaskImgRequest) isAnsweredTask() {}

func (t *AnsweredCheckedTextTaskImgRequest) Validate(ctx context.Context) *valgo.Validation {
	checkedTextReg := regexp.MustCompile(fmt.Sprintf("[%v]", configuration.CheckedTextAnswerTemplate))

	return t.BaseTaskWithImgRequest.Validate(ctx).
		Is(valgo.String(t.Type, "type", "type").EqualTo(CheckedText)).
		Is(valgo.String(t.Answer, "answer", "answer").
			MatchingTo(checkedTextReg).MaxLength(configuration.MaxCheckedTextAnswerLength))
}

type AnsweredChoiceTaskImgRequest struct {
	BaseTaskWithImgRequest

	Options     []string `json:"options"`
	AnswerIndex uint8    `json:"answer-idx"`
}

func (*AnsweredChoiceTaskImgRequest) isAnsweredTask() {}

func (t *AnsweredChoiceTaskImgRequest) Validate(ctx context.Context) *valgo.Validation {
	v := t.BaseTaskWithImgRequest.Validate(ctx).
		Is(valgo.String(t.Type, "type", "type").EqualTo(Choice)).
		Is(valgo.Uint8(t.AnswerIndex, "answer", "answer").LessThan(configuration.OptionsCount)).
		Is(valgo.Any(t.Options, "options", "options").Passing(func(v any) bool {
			return len(v.([]string)) == configuration.OptionsCount
		}))
	for i := 0; i < len(t.Options); i++ {
		v = v.Is(valgo.String(t.Options[i], "option", "option").MaxLength(configuration.MaxOptionLength))
	}
	return v
}

type AnsweredPhotoTaskImgRequest struct {
	BaseTaskWithImgRequest
}

func (*AnsweredPhotoTaskImgRequest) isAnsweredTask() {}

func (t *AnsweredPhotoTaskImgRequest) Validate(ctx context.Context) *valgo.Validation {
	return t.BaseTaskWithImgRequest.Validate(ctx).
		Is(valgo.String(t.Type, "type", "type").EqualTo(Photo)).
		Is(valgo.Any(t.PollDuration, "poll-duration", "poll-duration").Passing(func(v any) bool {
			d := v.(PollDuration)
			return d.Kind == Fixed || d.Kind == Dynamic
		}))
}

type AnsweredTextTaskImgRequest struct {
	BaseTaskWithImgRequest
}

func (*AnsweredTextTaskImgRequest) isAnsweredTask() {}

func (t *AnsweredTextTaskImgRequest) Validate(ctx context.Context) *valgo.Validation {
	return t.BaseTaskWithImgRequest.Validate(ctx).
		Is(valgo.String(t.Type, "type", "type").EqualTo(Text)).
		Is(valgo.Any(t.PollDuration, "poll-duration", "poll-duration").Passing(func(v any) bool {
			d := v.(PollDuration)
			return d.Kind == Fixed || d.Kind == Dynamic
		}))
}
