package api

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/cli/cli/internal/ghrepo"
)

type ActionsArtifact struct {
	Name               string `json:"name"`
	ArchiveDownloadUrl string `json:"archive_download_url"`
	// TODO: Add the rest of the fields
}

func GetActionsArtifacts(client *Client, repo ghrepo.Interface, runId uint64) ([]ActionsArtifact, error) {
	path := fmt.Sprintf("repos/%s/actions/runs/%d/artifacts", ghrepo.FullName(repo), runId)

	result := struct {
		Artifacts []ActionsArtifact `json:"artifacts"`
	}{}

	err := client.REST(repo.RepoHost(), "GET", path, nil, &result)
	if err != nil {
		return nil, err
	}

	// fmt.Println(repo.RepoHost())
	// fmt.Println(path)
	// fmt.Println(len(result.Artifacts))
	return result.Artifacts, nil
}

func GetSignedDownloadUrl(client *Client, downloadUrl string) (*url.URL, error) {
	httpClient := client.http

	originalCheckRedirect := httpClient.CheckRedirect
	defer func() {
		httpClient.CheckRedirect = originalCheckRedirect
	}()

	httpClient.CheckRedirect = checkRedirect

	resp, err := httpClient.Get(downloadUrl)
	if err != nil {
		return nil, err
	}

	return resp.Location()
}

func checkRedirect(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}
