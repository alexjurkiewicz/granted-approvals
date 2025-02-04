package users

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/common-fate/granted-approvals/pkg/cfaws"
	"github.com/common-fate/granted-approvals/pkg/clio"
	"github.com/common-fate/granted-approvals/pkg/deploy"
	"github.com/urfave/cli/v2"
)

var CreateCommand = cli.Command{
	Name: "create",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "username", Aliases: []string{"u"}, Usage: "The username of the user to create (should be an email address)", Required: true},
		&cli.BoolFlag{Name: "admin", Aliases: []string{"a"}, Usage: "Whether to make the user a Granted Approvals administrator"},
	},
	Description: "Create a Cognito user",
	Action: func(c *cli.Context) error {
		ctx := c.Context

		username := c.String("username")
		f := c.Path("file")
		dc, err := deploy.ConfigFromContext(ctx)
		if err != nil {
			return err
		}

		cfg, err := cfaws.ConfigFromContextOrDefault(ctx)
		if err != nil {
			return err
		}
		adminGroup := dc.Deployment.Parameters.AdministratorGroupID

		o, err := dc.LoadOutput(ctx)
		if err != nil {
			return err
		}
		cog := cognitoidentityprovider.NewFromConfig(cfg)
		_, err = cog.AdminCreateUser(ctx, &cognitoidentityprovider.AdminCreateUserInput{
			UserPoolId: &o.UserPoolID,
			Username:   &username,
		})
		if err != nil {
			return err
		}

		clio.Success("created user %s", username)

		if c.Bool("admin") {
			if adminGroup == "" {
				return clio.NewCLIError(fmt.Sprintf("The AdministratorGroupID parameter is not set in %s. Set the parameter in the Parameters section and then call 'gdeploy identity groups members add --username %s --group <the admin group ID>' to make the user a Granted administrator.", f, username))
			}

			_, err = cog.AdminAddUserToGroup(ctx, &cognitoidentityprovider.AdminAddUserToGroupInput{
				GroupName:  &adminGroup,
				Username:   &username,
				UserPoolId: &o.UserPoolID,
			})
			if err != nil {
				return err
			}

			clio.Success("added user %s to administrator group '%s'", username, adminGroup)
		}

		return nil
	},
}
