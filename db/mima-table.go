package db

type Mima interface {
	UpdatedAt() int64
	SetID(id string)
	SetPassword(password string)
	SetNotes(notes string)
	Seal(key *SecretKey) ([]byte, error)
}
