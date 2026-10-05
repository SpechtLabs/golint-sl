package a

// Good: user.pb.go matches a generated or mock file pattern and is skipped.
func skippedProto(u *User) string {
	return u.Name
}
