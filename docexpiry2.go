package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigateway"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"

	// "github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type Docexpiry2StackProps struct {
	awscdk.StackProps
}

func NewDocexpiry2Stack(scope constructs.Construct, id string, props *Docexpiry2StackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	table := awsdynamodb.NewTable(stack, jsii.String("tokenTable"), &awsdynamodb.TableProps{
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("access_token"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		TableName: jsii.String("Token"),
	})

	api := awsapigateway.NewRestApi(stack, jsii.String("docExpiryApiGateway"), &awsapigateway.RestApiProps{
		DefaultCorsPreflightOptions: &awsapigateway.CorsOptions{
			AllowHeaders: jsii.Strings(
				"Content-Type",
				"X-Amz-Date",
				"Authorization",
				"X-Api-Key",
				"X-Amz-Security-Token",
			),
			AllowMethods: jsii.Strings("GET", "POST", "PUT", "DELETE"),
			AllowOrigins: jsii.Strings("http://localhost:3000"),
		},
		DeployOptions: &awsapigateway.StageOptions{
	    LoggingLevel: awsapigateway.MethodLoggingLevel_INFO,
	},

	})
	
	myFunction := awslambda.NewFunction(stack, jsii.String("docExpiryLambdaFunc"), &awslambda.FunctionProps{
		Runtime:awslambda.Runtime_PROVIDED_AL2023() ,
		Code: awslambda.AssetCode_FromAsset(jsii.String("lambda/function.zip"), nil),
		Handler: jsii.String("main"),
	})
	table.GrantReadWriteData(myFunction)

	integration := awsapigateway.NewLambdaIntegration(myFunction, nil)
	loginResource := api.Root().AddResource(jsii.String("login"), nil)
	loginResource.AddMethod(jsii.String("GET"), integration, nil)

	callbackresource := api.Root().AddResource(jsii.String("oauth2callback"), nil)
	callbackresource.AddMethod(jsii.String("GET"), integration, nil)
	// The code that defines your stack goes here

	// example resource
	// queue := awssqs.NewQueue(stack, jsii.String("Docexpiry2Queue"), &awssqs.QueueProps{
	// 	VisibilityTimeout: awscdk.Duration_Seconds(jsii.Number(300)),
	// })

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	NewDocexpiry2Stack(app, "Docexpiry2Stack", &Docexpiry2StackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	// If unspecified, this stack will be "environment-agnostic".
	// Account/Region-dependent features and context lookups will not work, but a
	// single synthesized template can be deployed anywhere.
	//---------------------------------------------------------------------------
	return nil

	// Uncomment if you know exactly what account and region you want to deploy
	// the stack to. This is the recommendation for production stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String("123456789012"),
	//  Region:  jsii.String("us-east-1"),
	// }

	// Uncomment to specialize this stack for the AWS Account and Region that are
	// implied by the current CLI configuration. This is recommended for dev
	// stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
	//  Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	// }
}
