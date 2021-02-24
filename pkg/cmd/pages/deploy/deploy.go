package deploy

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/AlecAivazis/survey/v2"
	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/api"
	"github.com/cli/cli/internal/config"
	"github.com/cli/cli/internal/ghinstance"
	"github.com/cli/cli/internal/ghrepo"
	"github.com/cli/cli/pkg/cmdutil"
	"github.com/cli/cli/pkg/iostreams"
	"github.com/cli/cli/pkg/prompt"
	"github.com/spf13/cobra"
)

const PAGES_ARTIFACT_NAME = "gh-pages"

type DeployOptions struct {
	IO         *iostreams.IOStreams
	Config     func() (config.Config, error)
	HttpClient func() (*http.Client, error)
	BaseRepo   func() (ghrepo.Interface, error)

	Interactive bool

	SelectorArg string

	Hostname string
	Scopes   []string
	Token    string
	Web      bool
}

func NewCmdDeploy(f *cmdutil.Factory, runF func(*DeployOptions) error) *cobra.Command {
	opts := &DeployOptions{
		IO:         f.IOStreams,
		Config:     f.Config,
		HttpClient: f.HttpClient,
	}

	cmd := &cobra.Command{
		Use:   "deploy",
		Args:  cobra.ExactArgs(1),
		Short: "Deploy an artifact to GitHub Pages",
		Long: heredoc.Docf(`
			Deploy an artifact to GitHub Pages
		`, "`"),
		Example: heredoc.Doc(`
			# Deploy to pages with an actions workflow run artifact or an http url
			# Specify a run id
			$ gh pages deploy 42
			# Specify the current run id within an actions workflow
			$ echo ${{secrets.GITHUB_TOKEN}} | gh auth login --with-token
			$ gh pages deploy $GITHUB_RUN_ID
			# Specify a url
			$ gh pages deploy https://example.com/foo.zip
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.BaseRepo = f.BaseRepo

			if len(args) > 0 {
				opts.SelectorArg = args[0]
			}

			if runF != nil {
				return runF(opts)
			}

			return deployRun(opts)
		},
	}

	return cmd
}

func deployRun(opts *DeployOptions) error {
	_, err := opts.Config()
	if err != nil {
		return err
	}

	client, err := opts.HttpClient()
	if err != nil {
		return err
	}

	apiClient := api.NewClientFromHTTP(client)

	baseRepo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	fmt.Printf("Hello world: %s\n", opts.SelectorArg)

	var artifactURL *url.URL
	if runId, err := strconv.ParseUint(opts.SelectorArg, 10, 64); err == nil {
		downloadUrl, err := getArtifactUrl(apiClient, baseRepo, runId)
		if err != nil {
			return err
		}

		if artifactURL, err = api.GetSignedDownloadUrl(apiClient, *downloadUrl); err != nil {
			return err
		}
	} else {
		if artifactURL, err = url.Parse(opts.SelectorArg); err != nil {
			return err
		}

		if artifactURL.Scheme != "https" {
			return errors.New("Error: Only https is supported")
		}
	}

	fmt.Printf("Artifact url: %s %s\n", artifactURL.Scheme, artifactURL)

	err = api.Deploy(apiClient, baseRepo, *artifactURL)
	if err != nil {
		return err
	}

	fmt.Println("success?")
	return nil
}

func getArtifactUrl(apiClient *api.Client, repo ghrepo.Interface, runId uint64) (*string, error) {
	artifacts, err := api.GetActionsArtifacts(apiClient, repo, runId)
	if err != nil {
		return nil, err
	}

	for _, artifact := range artifacts {
		if artifact.Name == PAGES_ARTIFACT_NAME {
			return &artifact.ArchiveDownloadUrl, nil
		}
	}

	return nil, errors.New(fmt.Sprintf("Error: Artifact %q not found for runId %d", PAGES_ARTIFACT_NAME, runId))
}

func promptForHostname() (string, error) {
	var hostType int
	err := prompt.SurveyAskOne(&survey.Select{
		Message: "What account do you want to log into?",
		Options: []string{
			"GitHub.com",
			"GitHub Enterprise Server",
		},
	}, &hostType)

	if err != nil {
		return "", fmt.Errorf("could not prompt: %w", err)
	}

	isEnterprise := hostType == 1

	hostname := ghinstance.Default()
	if isEnterprise {
		err := prompt.SurveyAskOne(&survey.Input{
			Message: "GHE hostname:",
		}, &hostname, survey.WithValidator(ghinstance.HostnameValidator))
		if err != nil {
			return "", fmt.Errorf("could not prompt: %w", err)
		}
	}

	return hostname, nil
}
