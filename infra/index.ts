import * as pulumi from "@pulumi/pulumi";
import * as aws from "@pulumi/aws";
import * as path from "path";

const config = new pulumi.Config();
const stravaClientId = config.require("stravaClientId");
const stravaClientSecret = config.requireSecret("stravaClientSecret");

// IAM role for the Lambda function.
const role = new aws.iam.Role("strava-ap-proxy-role", {
  assumeRolePolicy: JSON.stringify({
    Version: "2012-10-17",
    Statement: [
      {
        Action: "sts:AssumeRole",
        Principal: { Service: "lambda.amazonaws.com" },
        Effect: "Allow",
      },
    ],
  }),
});

// Attach basic execution role (CloudWatch Logs only).
new aws.iam.RolePolicyAttachment("strava-ap-proxy-logs", {
  role: role.name,
  policyArn:
    "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole",
});

// Lambda function.
const fn = new aws.lambda.Function("strava-ap-proxy", {
  runtime: "nodejs20.x",
  handler: "index.handler",
  role: role.arn,
  code: new pulumi.asset.AssetArchive({
    "index.mjs": new pulumi.asset.FileAsset(
      path.join(__dirname, "lambda", "index.mjs")
    ),
  }),
  environment: {
    variables: {
      STRAVA_CLIENT_ID: stravaClientId,
      STRAVA_CLIENT_SECRET: stravaClientSecret,
    },
  },
  memorySize: 128,
  timeout: 10,
  reservedConcurrentExecutions: 1,
});

// Function URL (public HTTPS endpoint, no API Gateway needed).
const fnUrl = new aws.lambda.FunctionUrl("strava-ap-proxy-url", {
  functionName: fn.name,
  authorizationType: "NONE",
});

export const functionUrl = fnUrl.functionUrl;
