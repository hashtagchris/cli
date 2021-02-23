package api

import (
	"bytes"
	"fmt"

	"github.com/cli/cli/internal/ghrepo"
)

// Deploy takes an HTTPS URL of an artifact and deploys the site to Pages
func Deploy(client *Client, repo ghrepo.Interface, artifactURL string) error {
	path := fmt.Sprintf("repos/%s/pages", ghrepo.FullName(repo))
	body := bytes.NewBufferString(fmt.Sprintf(`{"remote_url": "%s"}`, artifactURL))
	var result interface{}
	err := client.REST(repo.RepoHost(), "PUT", path, body, &result)
	if err != nil {
		return err
	}

	return nil
}
