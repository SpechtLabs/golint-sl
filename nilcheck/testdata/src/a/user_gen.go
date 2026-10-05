package a

// Good: user_gen.go matches a generated or mock file pattern and is skipped.
func skippedGenSuffix(u *User) string {
	return u.Name
}
