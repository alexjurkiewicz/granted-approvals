package deploy

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go"
	"github.com/common-fate/granted-approvals/pkg/cfaws"
	"github.com/common-fate/granted-approvals/pkg/clio"
	"github.com/mitchellh/mapstructure"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
)

// Output is the output from deploying the CDK stack to AWS.
type Output struct {
	CognitoClientID               string `json:"CognitoClientID"`
	CloudFrontDomain              string `json:"CloudFrontDomain"`
	FrontendDomainOutput          string `json:"FrontendDomainOutput"`
	CloudFrontDistributionID      string `json:"CloudFrontDistributionID"`
	S3BucketName                  string `json:"S3BucketName"`
	UserPoolID                    string `json:"UserPoolID"`
	UserPoolDomain                string `json:"UserPoolDomain"`
	APIURL                        string `json:"APIURL"`
	WebhookURL                    string `json:"WebhookURL"`
	WebhookLogGroupName           string `json:"WebhookLogGroupName"`
	APILogGroupName               string `json:"APILogGroupName"`
	IDPSyncLogGroupName           string `json:"IDPSyncLogGroupName"`
	AccessHandlerLogGroupName     string `json:"AccessHandlerLogGroupName"`
	EventBusLogGroupName          string `json:"EventBusLogGroupName"`
	EventsHandlerLogGroupName     string `json:"EventsHandlerLogGroupName"`
	GranterLogGroupName           string `json:"GranterLogGroupName"`
	SlackNotifierLogGroupName     string `json:"SlackNotifierLogGroupName"`
	DynamoDBTable                 string `json:"DynamoDBTable"`
	GranterStateMachineArn        string `json:"GranterStateMachineArn"`
	EventBusArn                   string `json:"EventBusArn"`
	EventBusSource                string `json:"EventBusSource"`
	IdpSyncFunctionName           string `json:"IdpSyncFunctionName"`
	Region                        string `json:"Region"`
	PaginationKMSKeyARN           string `json:"PaginationKMSKeyARN"`
	AccessHandlerExecutionRoleARN string `json:"AccessHandlerExecutionRoleARN"`
}

func (c Output) FrontendURL() string {
	if c.FrontendDomainOutput == "" {
		return "https://" + c.CloudFrontDomain
	}

	return "https://" + c.FrontendDomainOutput
}

func (c Output) PrintTable() {
	v := reflect.ValueOf(c)
	t := v.Type()

	data := [][]string{}
	for i := 0; i < v.NumField(); i++ {
		val := fmt.Sprintf("%v", v.Field(i).Interface())
		data = append(data, []string{t.Field(i).Name, val})
	}

	table := tablewriter.NewWriter(os.Stderr)
	table.SetHeader([]string{"Output Parameter", "Value"})

	table.SetColumnColor(
		tablewriter.Colors{tablewriter.FgHiBlackColor},
		tablewriter.Colors{tablewriter.FgGreenColor, tablewriter.Bold},
	)

	for _, v := range data {
		table.Append(v)
	}
	table.Render()
}

// Keys returns the names of the output variables.
func (o Output) Keys() []string {
	v := reflect.ValueOf(o)
	t := v.Type()
	var keys []string

	for i := 0; i < v.NumField(); i++ {
		keys = append(keys, t.Field(i).Name)
	}
	return keys
}

// Get a value by it's key in the output struct
func (o Output) Get(key string) (string, error) {
	r := reflect.ValueOf(o)
	f := r.FieldByName(key)

	if !f.IsValid() {
		return "", fmt.Errorf("key %s not found in output", key)
	}

	return f.String(), nil
}

func (c SAMLOutputs) PrintSAMLTable() {
	data := [][]string{
		{"SAML SSO URL (ACS URL)", c.ACSURL},
		{"Audience URI", c.AudienceURI},
	}

	table := tablewriter.NewWriter(os.Stderr)
	table.SetHeader([]string{"Output Parameter", "Value"})

	table.SetColumnColor(
		tablewriter.Colors{tablewriter.FgHiBlackColor},
		tablewriter.Colors{tablewriter.FgGreenColor, tablewriter.Bold},
	)

	for _, v := range data {
		table.Append(v)
	}
	table.Render()
}

type SAMLOutputs struct {
	// ACS URL
	ACSURL      string
	AudienceURI string
}

// LoadOutput loads the outputs for the current deployment.
func (c *Config) LoadSAMLOutput(ctx context.Context) (SAMLOutputs, error) {
	if c.cachedSAMLOutput != nil {
		return *c.cachedSAMLOutput, nil
	}
	cfg, err := cfaws.ConfigFromContextOrDefault(ctx)
	if err != nil {
		return SAMLOutputs{}, err
	}
	client := cloudformation.NewFromConfig(cfg)
	res, err := client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: &c.Deployment.StackName,
	})

	var ve *smithy.GenericAPIError
	if errors.As(err, &ve) && ve.Code == "ValidationError" {
		clio.Error("We couldn't find a CloudFormation stack '%s' in region '%s'.", c.Deployment.StackName, c.Deployment.Region)
		clio.Log(`
To fix this, take one of the following actions:
  a) verify that your AWS credentials match the account you're trying to deploy to (%s). You can check this by calling 'aws sts get-caller-identity'.
  b) your stack may not have been deployed yet. Run 'gdeploy create' to deploy it using CloudFormation.
`, c.Deployment.Account)
		return SAMLOutputs{}, err
	}

	if err != nil {
		return SAMLOutputs{}, err
	}

	if len(res.Stacks) != 1 {
		return SAMLOutputs{}, fmt.Errorf("expected 1 stack but got %d", len(res.Stacks))
	}

	stack := res.Stacks[0]
	out := SAMLOutputs{}

	for _, o := range stack.Outputs {
		if *o.OutputKey == "UserPoolDomain" {
			out.ACSURL = "https://" + *o.OutputValue + "/saml2/idpresponse"
		}
		if *o.OutputKey == "UserPoolID" {
			out.AudienceURI = fmt.Sprintf("urn:amazon:cognito:sp:%s", *o.OutputValue)
		}
	}
	// set the cached output value in case this method is called again.
	c.cachedSAMLOutput = &out
	return out, nil
}

// GetStackStatus indicates whether the Cloud Formation stack is online (via "CREATE_COMPLETE")
func (c *Config) GetStackStatus(ctx context.Context) (types.StackStatus, error) {
	cfg, err := cfaws.ConfigFromContextOrDefault(ctx)
	if err != nil {
		return "", err
	}
	client := cloudformation.NewFromConfig(cfg)
	res, err := client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: &c.Deployment.StackName,
	})
	if err != nil {
		return "", err
	}
	if len(res.Stacks) != 1 {
		return "", fmt.Errorf("expected 1 stack but got %d", len(res.Stacks))
	}

	stack := res.Stacks[0]

	return stack.StackStatus, nil
}

// LoadOutput loads the outputs for the current deployment.
func (c *Config) LoadOutput(ctx context.Context) (Output, error) {
	if c.cachedOutput != nil {
		return *c.cachedOutput, nil
	}

	cfg, err := cfaws.ConfigFromContextOrDefault(ctx)
	if err != nil {
		return Output{}, err
	}
	client := cloudformation.NewFromConfig(cfg)
	res, err := client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: &c.Deployment.StackName,
	})

	var ve *smithy.GenericAPIError
	if errors.As(err, &ve) && ve.Code == "ValidationError" {
		clio.Error("We couldn't find a CloudFormation stack '%s' in region '%s'.", c.Deployment.StackName, c.Deployment.Region)
		clio.Log(`
To fix this, take one of the following actions:
  a) verify that your AWS credentials match the account you're trying to deploy to (%s). You can check this by calling 'aws sts get-caller-identity'.
  b) your stack may not have been deployed yet. Run 'gdeploy create' to deploy it using CloudFormation.
`, c.Deployment.Account)
		return Output{}, err
	}

	if err != nil {
		return Output{}, err
	}

	if len(res.Stacks) != 1 {
		return Output{}, fmt.Errorf("expected 1 stack but got %d", len(res.Stacks))
	}

	stack := res.Stacks[0]

	// decode the output variables into the Go struct.
	outputMap := make(map[string]string)
	for _, o := range stack.Outputs {
		outputMap[*o.OutputKey] = *o.OutputValue
	}

	var out Output
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{TagName: "json", Result: &out})
	if err != nil {
		return Output{}, errors.Wrap(err, "setting up decoder")
	}
	err = decoder.Decode(outputMap)
	if err != nil {
		return Output{}, errors.Wrap(err, "decoding CloudFormation outputs")
	}

	// set the cached output value in case this method is called again.
	c.cachedOutput = &out

	return out, nil
}
