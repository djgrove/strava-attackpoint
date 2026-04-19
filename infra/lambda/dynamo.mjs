import { DynamoDBClient } from "@aws-sdk/client-dynamodb";
import {
  DynamoDBDocumentClient,
  GetCommand,
  PutCommand,
  DeleteCommand,
  UpdateCommand,
} from "@aws-sdk/lib-dynamodb";

const client = new DynamoDBClient({});
const docClient = DynamoDBDocumentClient.from(client);
const TABLE = process.env.DYNAMODB_TABLE;

export async function getUser(stravaAthleteId) {
  const resp = await docClient.send(
    new GetCommand({
      TableName: TABLE,
      Key: { strava_athlete_id: String(stravaAthleteId) },
    })
  );
  return resp.Item || null;
}

export async function putUser(data) {
  await docClient.send(
    new PutCommand({
      TableName: TABLE,
      Item: {
        ...data,
        strava_athlete_id: String(data.strava_athlete_id),
        registered_at: new Date().toISOString(),
        status: "active",
      },
    })
  );
}

export async function deleteUser(stravaAthleteId) {
  await docClient.send(
    new DeleteCommand({
      TableName: TABLE,
      Key: { strava_athlete_id: String(stravaAthleteId) },
    })
  );
}

export async function updateTokens(stravaAthleteId, accessToken, refreshToken, expiresAt) {
  await docClient.send(
    new UpdateCommand({
      TableName: TABLE,
      Key: { strava_athlete_id: String(stravaAthleteId) },
      UpdateExpression:
        "SET strava_access_token = :at, strava_refresh_token = :rt, strava_token_expires_at = :exp",
      ExpressionAttributeValues: {
        ":at": accessToken,
        ":rt": refreshToken,
        ":exp": expiresAt,
      },
    })
  );
}

export async function updateLastSync(stravaAthleteId) {
  await docClient.send(
    new UpdateCommand({
      TableName: TABLE,
      Key: { strava_athlete_id: String(stravaAthleteId) },
      UpdateExpression: "SET last_sync_at = :ts",
      ExpressionAttributeValues: { ":ts": new Date().toISOString() },
    })
  );
}

export async function updateStatus(stravaAthleteId, status) {
  await docClient.send(
    new UpdateCommand({
      TableName: TABLE,
      Key: { strava_athlete_id: String(stravaAthleteId) },
      UpdateExpression: "SET #s = :s",
      ExpressionAttributeNames: { "#s": "status" },
      ExpressionAttributeValues: { ":s": status },
    })
  );
}
