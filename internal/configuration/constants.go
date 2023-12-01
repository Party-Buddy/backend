package configuration

const (
	PlayerMin int8 = 2
	PlayerMax int8 = 20

	BaseTextFieldTemplate string = "a-zA-Zа-яА-Я0-9,./?<>()\\-_+=|;:!@#$%^&*{}\\[\\]\"'\\\\№`~ "

	MaxNameLength        = 20
	MaxDescriptionLength = 255

	MinTaskCount = 1
	MaxTaskCount = 100

	MaxCheckedTextAnswerLength        = 20
	CheckedTextAnswerTemplate  string = "A-ZА-Я0-9,./?<>()\\-_+=|;:!@#$%^&*{}\\[\\]\"'\\\\№`~ "

	OptionsCount    = 4
	MaxOptionLength = 20

	MaxTextAnswerLength = 255
)
