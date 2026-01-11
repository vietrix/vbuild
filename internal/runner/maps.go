package runner

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
