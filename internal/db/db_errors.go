package db

type RecordNotFound struct{}

func (r RecordNotFound) Error() string {
	return "record-not-found"
}
