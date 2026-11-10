package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"paperly/locales"

	"github.com/egoist/mygo"
)

// The project's home on GitHub, and the endpoint that names the newest
// release: the update check asks the latter about the former.
const (
	repoURL     = "https://github.com/jiangwanying/paperly"
	releasesURL = repoURL + "/releases/latest"
	releaseAPI  = "https://api.github.com/repos/jiangwanying/paperly/releases/latest"
)

// checkForUpdates asks GitHub which release is newest and tells the user,
// on a thread the dialog is allowed to run on. The dialog's choices act at
// once: a newer release opens its download page, the rest just close.
func checkForUpdates() {
	go func() {
		latest, err := fetchLatestRelease()
		switch {
		case err != nil:
			updateDialog("Paperly", locales.UpdateFailed(), locales.UpdateFailedDetail(err.Error()),
				[]string{locales.Ok()}, func(int) {})
		case newerThan(version, latest):
			updateDialog("Paperly", locales.UpdateFound(latest), locales.UpdateFoundDetail(version, latest),
				[]string{locales.OpenDownloadPage(), locales.Ok()},
				func(b int) {
					if b == 0 {
						openURL(releasesURL)
					}
				})
		default:
			updateDialog("Paperly", locales.UpToDate(), locales.UpToDateDetail(version),
				[]string{locales.Ok()}, func(int) {})
		}
	}()
}

// updateDialog shows a dialog of the update check, and hands the chosen
// button's index to onChoose once it closes.
func updateDialog(title, message, detail string, buttons []string, onChoose func(int)) {
	go func() {
		r, err := mygo.Dialog.Message(mygo.MessageOptions{
			Title:   title,
			Message: message,
			Detail:  detail,
			Buttons: buttons,
		})
		if err != nil {
			return
		}
		onChoose(r.Button)
	}()
}

// fetchLatestRelease asks the release API which version is newest. The
// answer's tag reads v0.1.1; only the numbers matter.
func fetchLatestRelease() (string, error) {
	req, err := http.NewRequest(http.MethodGet, releaseAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "paperly-"+version)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.TagName == "" {
		return "", fmt.Errorf("%s", "the release names no version")
	}
	return strings.TrimPrefix(strings.TrimSpace(body.TagName), "v"), nil
}

// newerThan reports whether latest is a newer release than the build's own
// version: the three numbers count up, anything unreadable does not count.
func newerThan(current, latest string) bool {
	c, ok := parseSemver(current)
	if !ok {
		return false
	}
	l, ok := parseSemver(latest)
	if !ok {
		return false
	}
	return l[0] > c[0] || (l[0] == c[0] && (l[1] > c[1] ||
		(l[1] == c[1] && l[2] > c[2])))
}

// parseSemver reads "v1.2.3", "1.2.3" and the like into the three numbers;
// anything else is not a version.
func parseSemver(s string) ([3]int, bool) {
	var out [3]int
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if s == "" {
		return out, false
	}
	parts := strings.Split(s, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// openURL hands an address to the system, which opens it with whatever the
// platform opens addresses with.
func openURL(url string) {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", url).Start()
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
	}
}
