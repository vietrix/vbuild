package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type release struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

func fetchRelease(toVersion string) (*release, error) {
	endpoint := "https://api.github.com/repos/vietrix/vbuild/releases/latest"
	if toVersion != "" {
		endpoint = fmt.Sprintf("https://api.github.com/repos/vietrix/vbuild/releases/tags/%s", toVersion)
	}

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "vbuild")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch release: %s", resp.Status)
	}

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	if rel.TagName == "" {
		return nil, errors.New("release missing tag name")
	}
	return &rel, nil
}
