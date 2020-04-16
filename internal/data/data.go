package data

// UserProfile is a structure for a person
type UserProfile struct {
	UID   string `json:"uid"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}
