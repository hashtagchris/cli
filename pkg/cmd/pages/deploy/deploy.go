package deploy

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

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

	var tokenStdin bool

	cmd := &cobra.Command{
		Use:   "deploy",
		Args:  cobra.ExactArgs(1),
		Short: "Authenticate with a GitHub host",
		Long: heredoc.Docf(`
			Authenticate with a GitHub host.

			The default authentication mode is a web-based browser flow.

			Alternatively, pass in a token on standard input by using %[1]s--with-token%[1]s.
			The minimum required scopes for the token are: "repo", "read:org".

			The --scopes flag accepts a comma separated list of scopes you want your gh credentials to have. If
			absent, this command ensures that gh has access to a minimum set of scopes.
		`, "`"),
		Example: heredoc.Doc(`
			# start interactive setup
			$ gh auth login

			# authenticate against github.com by reading the token from a file
			$ gh auth login --with-token < mytoken.txt

			# authenticate with a specific GitHub Enterprise Server instance
			$ gh auth login --hostname enterprise.internal
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !opts.IO.CanPrompt() && !(tokenStdin || opts.Web) {
				return &cmdutil.FlagError{Err: errors.New("--web or --with-token required when not running interactively")}
			}

			opts.BaseRepo = f.BaseRepo

			if len(args) > 0 {
				opts.ArtifactUrl = args[0]
			}

			if tokenStdin && opts.Web {
				return &cmdutil.FlagError{Err: errors.New("specify only one of --web or --with-token")}
			}

			if tokenStdin {
				defer opts.IO.In.Close()
				token, err := ioutil.ReadAll(opts.IO.In)
				if err != nil {
					return fmt.Errorf("failed to read token from STDIN: %w", err)
				}
				opts.Token = strings.TrimSpace(string(token))
			}

			if opts.IO.CanPrompt() && opts.Token == "" && !opts.Web {
				opts.Interactive = true
			}

			if cmd.Flags().Changed("hostname") {
				if err := ghinstance.HostnameValidator(opts.Hostname); err != nil {
					return &cmdutil.FlagError{Err: fmt.Errorf("error parsing --hostname: %w", err)}
				}
			}

			if !opts.Interactive {
				if opts.Hostname == "" {
					opts.Hostname = ghinstance.Default()
				}
			}

			if runF != nil {
				return runF(opts)
			}

			return loginRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Hostname, "hostname", "h", "", "The hostname of the GitHub instance to authenticate with")
	cmd.Flags().StringSliceVarP(&opts.Scopes, "scopes", "s", nil, "Additional authentication scopes for gh to have")
	cmd.Flags().BoolVar(&tokenStdin, "with-token", false, "Read token from standard input")
	cmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Open a browser to authenticate")

	return cmd
}

func loginRun(opts *DeployOptions) error {
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
