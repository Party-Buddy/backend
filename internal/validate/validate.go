package validate

import (
	"context"

	"github.com/cohesivestack/valgo"
)

type Validator interface {
	Validate(ctx context.Context) *valgo.Validation
}

func NewValidationFactory() *valgo.ValidationFactory {
	locales := make(map[string]*valgo.Locale)
	locales["en"] = &valgo.Locale{
		ErrorKeyFieldSet: "{{name}} must be provided",
	}
	return valgo.Factory(valgo.FactoryOptions{
		LocaleCodeDefault: "en",
		Locales:           locales,
	})
}

type key struct{}

var factoryKey key

func NewContext(ctx context.Context, f *valgo.ValidationFactory) context.Context {
	return context.WithValue(ctx, factoryKey, f)
}

func FromContext(ctx context.Context) (*valgo.ValidationFactory, bool) {
	f, ok := ctx.Value(factoryKey).(*valgo.ValidationFactory)
	return f, ok
}

const ErrorKeyFieldSet = "pb/field_set"

type ValidatorField[T any] struct {
	context *valgo.ValidatorContext
}

func (v *ValidatorField[T]) Context() *valgo.ValidatorContext {
	return v.context
}

func (v *ValidatorField[T]) Set(template ...string) *ValidatorField[T] {
	v.context.Add(
		func() bool {
			v, ok := v.context.Value().(*T)
			return ok && v != nil
		},
		ErrorKeyFieldSet,
		template...,
	)

	return v
}

func (v *ValidatorField[T]) Not() *ValidatorField[T] {
	v.context.Not()

	return v
}

func FieldValue[T any](value *T, nameAndTitle ...string) *ValidatorField[T] {
	return &ValidatorField[T]{context: valgo.NewContext(value, nameAndTitle...)}
}

func ExtractValgoErrorFields(err *valgo.Error) (fieldName string, msg string, ok bool) {
	if err == nil {
		return
	}

	for _, v := range err.Errors() {
		if len(v.Messages()) > 0 {
			return v.Name(), v.Messages()[0], true
		}
	}

	return
}
