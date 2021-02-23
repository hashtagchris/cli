package deploy

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

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

type DeployOptions struct {
	IO         *iostreams.IOStreams
	Config     func() (config.Config, error)
	HttpClient func() (*http.Client, error)
	BaseRepo   func() (ghrepo.Interface, error)

	Interactive bool

	ArtifactUrl string

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
			# Deploy to pages with an HTTP url
			$ gh pages deploy https://example.com/foo.zip
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.BaseRepo = f.BaseRepo

			if len(args) > 0 {
				opts.ArtifactUrl = args[0]
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

	fmt.Printf("Hello world: %s\n", opts.ArtifactUrl)

	var artifactURL *url.URL
	if artifactURL, err = url.Parse(opts.ArtifactUrl); err != nil {
		return err
	}

	if artifactURL.Scheme != "https" {
		return errors.New("Error: Only https is supported")
	}

	fmt.Printf("Artifact url: %s %s\n", artifactURL.Scheme, artifactURL)

	client, err := opts.HttpClient()
	if err != nil {
		return err
	}

	apiClient := api.NewClientFromHTTP(client)

	baseRepo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	err = api.Deploy(apiClient, baseRepo, opts.ArtifactUrl)
	if err != nil {
		return err
	}

	fmt.Println("success?")
	return nil

	// hostname := opts.Hostname
	// if hostname == "" {
	// 	if opts.Interactive {
	// 		var err error
	// 		hostname, err = promptForHostname()
	// 		if err != nil {
	// 			return err
	// 		}
	// 	} else {
	// 		return errors.New("must specify --hostname")
	// 	}
	// }

	// if err := cfg.CheckWriteable(hostname, "oauth_token"); err != nil {
	// 	var roErr *config.ReadOnlyEnvError
	// 	if errors.As(err, &roErr) {
	// 		fmt.Fprintf(opts.IO.ErrOut, "The value of the %s environment variable is being used for authentication.\n", roErr.Variable)
	// 		fmt.Fprint(opts.IO.ErrOut, "To have GitHub CLI store credentials instead, first clear the value from the environment.\n")
	// 		return cmdutil.SilentError
	// 	}
	// 	return err
	// }

	// httpClient, err := opts.HttpClient()
	// if err != nil {
	// 	return err
	// }

	// if opts.Token != "" {
	// 	err := cfg.Set(hostname, "oauth_token", opts.Token)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	if err := shared.HasMinimumScopes(httpClient, hostname, opts.Token); err != nil {
	// 		return fmt.Errorf("error validating token: %w", err)
	// 	}

	// 	return cfg.Write()
	// }

	// existingToken, _ := cfg.Get(hostname, "oauth_token")
	// if existingToken != "" && opts.Interactive {
	// 	if err := shared.HasMinimumScopes(httpClient, hostname, existingToken); err == nil {
	// 		var keepGoing bool
	// 		err = prompt.SurveyAskOne(&survey.Confirm{
	// 			Message: fmt.Sprintf(
	// 				"You're already logged into %s. Do you want to re-authenticate?",
	// 				hostname),
	// 			Default: false,
	// 		}, &keepGoing)
	// 		if err != nil {
	// 			return fmt.Errorf("could not prompt: %w", err)
	// 		}
	// 		if !keepGoing {
	// 			return nil
	// 		}
	// 	}
	// }

	// return shared.Login(&shared.LoginOptions{
	// 	IO:          opts.IO,
	// 	Config:      cfg,
	// 	HTTPClient:  httpClient,
	// 	Hostname:    hostname,
	// 	Interactive: opts.Interactive,
	// 	Web:         opts.Web,
	// 	Scopes:      opts.Scopes,
	// })
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
