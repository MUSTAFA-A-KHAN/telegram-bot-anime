package dictionary

type Dictionary interface {
	GetMeaning(word string) (string, error)
}
