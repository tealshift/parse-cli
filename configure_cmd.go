package main

import (
	"fmt"
	"io"
	"os"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type configureCmd struct {
	login       login
	isDefault   bool
	tokenReader io.Reader // for testing
}

func (c *configureCmd) accountKey(e *env) error {
	token, err := c.login.helpCreateToken(e)
	if err != nil {
		return err
	}

	email, err := c.login.authToken(e, token)
	if err != nil {
		fmt.Fprintln(e.Err, "Could not store credentials. Please try again.\n")
		return err
	}

	if c.isDefault {
		email = ""
	}

	var l login
	if c.tokenReader != nil {
		l.tokenReader = c.tokenReader
	}
	foundEmail, creds, err := l.getTokenCredentials(e, email)
	if stackerr.HasUnderlying(err, stackerr.MatcherFunc(os.IsNotExist)) && !c.isDefault {
		fmt.Fprintln(
			e.Out,
			`

Looks like you have not configured the default account key yet.
Note that "parse new" and "parse list" can automatically pick up a default key if present.
Otherwise, you'll have to explicitly set the PARSER_EMAIL environment variable
for it to know which account key to use.
Further, if the command line tool cannot find an account key for a configured email it will try to
use the default account key

To configure the default account key use:
       "parse configure accountkey -d"`,
		)
	}

	if creds != nil {
		if c.isDefault {
			fmt.Fprintln(
				e.Err,
				"Note: this operation will overwrite the default account key",
			)
		} else if foundEmail {
			fmt.Fprintf(
				e.Err,
				`Note: this operation will overwrite the account key:
 %q
for email: %q
`,
				last4(token),
				email,
			)
		}
	}

	err = c.login.storeCredentials(e, email, &credentials{token: token})
	if err == nil {
		if c.isDefault {
			fmt.Fprintln(e.Out, "Successfully stored default account key.")
		} else {
			fmt.Fprintf(e.Out, "Successfully stored account key for: %q.\n", email)
		}
	}
	return stackerr.Wrap(err)
}

func (c *configureCmd) parserEmail(e *env, args []string) error {
	config, err := configFromDir(e.Root)
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return fmt.Errorf("Invalid args: %v, only an email argument is expected.", args)
	}
	config.getProjectConfig().ParserEmail = args[0]
	err = storeProjectConfig(e, config)
	if err != nil {
		fmt.Fprintln(e.Err, "Could not set parser email for project.")
		return err
	}
	fmt.Fprintf(e.Out, "Successfully configured email for current project to: %q\n", args[0])
	return nil
}

func newConfigureCmd(e *env) *cobra.Command {
	var c configureCmd

	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure various Parse settings",
		Long:  "Configure various Parse settings like account keys, project type, and more.",
		Run: func(c *cobra.Command, args []string) {
			c.Help()
		},
	}

	keyCmd := &cobra.Command{
		Use:     "accountkey",
		Short:   "Store Parse account key on machine",
		Long:    "Stores Parse account key in ~/.parse/netrc.",
		Run:     runNoArgs(e, c.accountKey),
		Aliases: []string{"key"},
	}
	keyCmd.Flags().BoolVarP(&c.isDefault, "default", "d", c.isDefault,
		"Make this token a system default")
	cmd.AddCommand(keyCmd)

	emailCmd := &cobra.Command{
		Use:   "email",
		Short: "Configures the parser email for this project",
		Long:  "Configures the parser email for current project.",
		Run:   runWithArgs(e, c.parserEmail),
	}
	cmd.AddCommand(emailCmd)

	return cmd
}