package a

// Good: zz_generated.deepcopy.go matches a generated or mock file pattern and is skipped.
func skippedDeepcopy(u *User) string {
	return u.Name
}
