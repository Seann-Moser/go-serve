package clientpkg

import "fmt"

func GetFlagWithPrefix(flag, prefix string) string {
	if prefix == "" {
		return flag
	}
	return fmt.Sprintf("%s-%s", prefix, flag)
}

func MergeMap[T any](m1, m2 map[string]T) map[string]T {
	if m1 == nil {
		return m2
	}
	if m2 == nil {
		return m1
	}
	if m1 == nil && m2 == nil {
		return map[string]T{}
	}
	for k, v := range m2 {
		if _, found := m1[k]; found {
			continue
		}
		m1[k] = v
	}
	return m1
}
