package archive

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const releasesAPIURL = "https://api.github.com/repos/sentzunhat/mochila-archive-viewer/releases/latest"

type UpdateStatus struct {
	Available bool   `json:"available"`
	Latest    string `json:"latest"`
	URL       string `json:"url"`
}

// CheckForUpdate compares currentVersion against the latest published GitHub
// Release. Never returns an error to the caller — any failure (offline, no
// releases yet, rate limited, malformed response) is treated as "no update
// available" so this can never surface a scary error to a local-first app
// that's expected to work without network access.
func CheckForUpdate(currentVersion string) UpdateStatus {
	none := UpdateStatus{Available: false}
	if currentVersion == "" || currentVersion == "dev" {
		return none // unreleased/dev build — never nag about updating
	}

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest("GET", releasesAPIURL, nil)
	if err != nil {
		return none
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return none // offline, DNS failure, timeout — all land here
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return none // 404 (no releases yet), 403 (rate limited), etc.
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return none
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")
	if latest == "" || !isNewer(latest, current) {
		return none
	}
	return UpdateStatus{Available: true, Latest: release.TagName, URL: release.HTMLURL}
}

// isNewer does a simple dotted-numeric compare (e.g. "0.2.0" > "0.1.0").
// Not a full semver parser (no pre-release/build metadata handling) —
// sufficient for this project's tag scheme (see release.yml: v0.1.0 style).
func isNewer(a, b string) bool {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		var av, bv int
		if i < len(as) {
			av = atoiSafe(as[i])
		}
		if i < len(bs) {
			bv = atoiSafe(bs[i])
		}
		if av != bv {
			return av > bv
		}
	}
	return false
}

func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}
