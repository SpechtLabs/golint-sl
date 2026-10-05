package a

// Good: mock_user.go matches a generated or mock file pattern and is skipped.
func skippedMock(u *User) string {
	return u.Name
}
