// Package mocks is under a mocks/ directory and is skipped entirely.
package mocks

type User struct{ Name string }

func Use(u *User) string {
	return u.Name
}
