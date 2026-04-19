import * as pulumi from "@pulumi/pulumi";
import * as aws from "@pulumi/aws";
import * as path from "path";

const config = new pulumi.Config();
const stravaClientId = config.require("stravaClientId");
const stravaClientSecret = config.requireSecret("stravaClientSecret");
const webhookVerifyToken = config.requireSecret("webhookVerifyToken");

// DynamoDB table for auto-sync users.
const usersTable = new aws.dynamodb.Table("strava-ap-users", {
  attributes: [{ name: "strava_athlete_id", type: "S" }],
  hashKey: "strava_athlete_id",
  billingMode: "PAY_PER_REQUEST",
});

// SQS dead-letter queue for failed webhook processing.
const dlq = new aws.sqs.Queue("strava-ap-webhook-dlq", {
  messageRetentionSeconds: 1209600, // 14 days
});

// SQS queue for webhook events.
const webhookQueue = new aws.sqs.Queue("strava-ap-webhook-queue", {
  visibilityTimeoutSeconds: 150, // > Lambda timeout (120s)
  redrivePolicy: pulumi.jsonStringify({
    maxReceiveCount: 3,
    deadLetterTargetArn: dlq.arn,
  }),
});

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

// DynamoDB + SQS permissions.
new aws.iam.RolePolicy("strava-ap-proxy-dynamo-sqs", {
  role: role.name,
  policy: pulumi.jsonStringify({
    Version: "2012-10-17",
    Statement: [
      {
        Effect: "Allow",
        Action: [
          "dynamodb:GetItem",
          "dynamodb:PutItem",
          "dynamodb:DeleteItem",
          "dynamodb:UpdateItem",
        ],
        Resource: usersTable.arn,
      },
      {
        Effect: "Allow",
        Action: [
          "sqs:SendMessage",
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes",
        ],
        Resource: webhookQueue.arn,
      },
    ],
  }),
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
    "dynamo.mjs": new pulumi.asset.FileAsset(
      path.join(__dirname, "lambda", "dynamo.mjs")
    ),
  }),
  environment: {
    variables: {
      STRAVA_CLIENT_ID: stravaClientId,
      STRAVA_CLIENT_SECRET: stravaClientSecret,
      STRAVA_WEBHOOK_VERIFY_TOKEN: webhookVerifyToken,
      DYNAMODB_TABLE: usersTable.name,
      SQS_QUEUE_URL: webhookQueue.url,
    },
  },
  memorySize: 256,
  timeout: 120,
  reservedConcurrentExecutions: 1,
});

// SQS event source mapping — triggers Lambda from webhook queue.
new aws.lambda.EventSourceMapping("strava-ap-sqs-trigger", {
  eventSourceArn: webhookQueue.arn,
  functionName: fn.name,
  batchSize: 1,
});

// API Gateway HTTP API.
const api = new aws.apigatewayv2.Api("strava-ap-api", {
  protocolType: "HTTP",
  corsConfiguration: {
    allowOrigins: ["*"],
    allowMethods: ["POST", "GET", "OPTIONS"],
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

// POST catch-all route.
new aws.apigatewayv2.Route("strava-ap-route", {
  apiId: api.id,
  routeKey: "POST /{proxy+}",
  target: pulumi.interpolate`integrations/${integration.id}`,
});

// GET catch-all route (for Strava webhook validation).
new aws.apigatewayv2.Route("strava-ap-get-route", {
  apiId: api.id,
  routeKey: "GET /{proxy+}",
  target: pulumi.interpolate`integrations/${integration.id}`,
});

// Default stage with auto-deploy.
new aws.apigatewayv2.Stage("strava-ap-stage", {
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
