package validate

import (
	"context"

	"github.com/cohesivestack/valgo"
)

type Validator interface {
	Validate(ctx context.Context) *valgo.Validation
}

func NewValidationFactory() *valgo.ValidationFactory {
	return valgo.Factory(valgo.FactoryOptions{
		LocaleCodeDefault: "en",
		Locales: map[string]*valgo.Locale{
			"en": {
				ErrorKeyFieldSet: "{{title}} should be set",
			},
		},
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

type ValidatorField struct {
	context *valgo.ValidatorContext
}

func (v *ValidatorField) Context() *valgo.ValidatorContext {
	return v.context
}

func (v *ValidatorField) Set(template ...string) *ValidatorField {
	v.context.Add(
		func() bool {
			return v.context.Value() != nil
		},
		ErrorKeyFieldSet,
		template...,
	)

	return v
}

func (v *ValidatorField) Not() *ValidatorField {
	v.context.Not()

	return v
}

func FieldValue[T any](value *T, nameAndTitle ...string) *ValidatorField {
	return &ValidatorField{context: valgo.NewContext(value, nameAndTitle...)}
}
