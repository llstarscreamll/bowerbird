package v1

func nullIfEmptyStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
