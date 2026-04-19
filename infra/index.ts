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
    "strava.mjs": new pulumi.asset.FileAsset(
      path.join(__dirname, "lambda", "strava.mjs")
    ),
    "attackpoint.mjs": new pulumi.asset.FileAsset(
      path.join(__dirname, "lambda", "attackpoint.mjs")
    ),
    "mapping.mjs": new pulumi.asset.FileAsset(
      path.join(__dirname, "lambda", "mapping.mjs")
    ),
  }),
  environment: {
    variables: {
      STRAVA_CLIENT_ID: stravaClientId,
      STRAVA_CLIENT_SECRET: stravaClientSecret,
    },
  },
  memorySize: 256,
  timeout: 120,
  reservedConcurrentExecutions: 1,
});

// API Gateway HTTP API (replaces Function URL which has auth issues).
const api = new aws.apigatewayv2.Api("strava-ap-api", {
  protocolType: "HTTP",
  corsConfiguration: {
    allowOrigins: ["*"],
    allowMethods: ["POST", "OPTIONS"],
    allowHeaders: ["Content-Type"],
  },
});

// Lambda integration.
const integration = new aws.apigatewayv2.Integration("strava-ap-integration", {
  apiId: api.id,
  integrationType: "AWS_PROXY",
  integrationUri: fn.arn,
  payloadFormatVersion: "2.0",
});

// Catch-all route.
const route = new aws.apigatewayv2.Route("strava-ap-route", {
  apiId: api.id,
  routeKey: "POST /{proxy+}",
  target: pulumi.interpolate`integrations/${integration.id}`,
});

// Default stage with auto-deploy.
const stage = new aws.apigatewayv2.Stage("strava-ap-stage", {
  apiId: api.id,
  name: "$default",
  autoDeploy: true,
});

// Allow API Gateway to invoke the Lambda.
new aws.lambda.Permission("strava-ap-apigw", {
  action: "lambda:InvokeFunction",
  function: fn.name,
  principal: "apigateway.amazonaws.com",
  sourceArn: pulumi.interpolate`${api.executionArn}/*/*`,
});

export const apiUrl = api.apiEndpoint;
