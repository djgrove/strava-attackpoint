const BASE_URL = "https://www.strava.com/api/v3";

export async function fetchActivities(accessToken, since, end) {
  const activities = [];
  let page = 1;
  const after = Math.floor(new Date(since).getTime() / 1000);
  const before = end ? Math.floor(new Date(end).getTime() / 1000) : undefined;

  while (true) {
    let url = `${BASE_URL}/athlete/activities?after=${after}&page=${page}&per_page=100`;
    if (before) url += `&before=${before}`;

    const resp = await fetch(url, {
      headers: { Authorization: `Bearer ${accessToken}` },
    });

    if (resp.status === 401) throw new Error("Strava token expired");
    if (!resp.ok) throw new Error(`Strava API error: ${resp.status}`);

    const data = await resp.json();
    if (data.length === 0) break;

    activities.push(...data);
    page++;
  }

  return activities;
}

export async function fetchActivityDetail(accessToken, activityId) {
  const resp = await fetch(`${BASE_URL}/activities/${activityId}`, {
    headers: { Authorization: `Bearer ${accessToken}` },
  });

  if (!resp.ok) return null;
  return resp.json();
}

export function estimateIntensity(avgHR, maxHR) {
  if (!avgHR || !maxHR || maxHR === 0) return 0;
  const ratio = avgHR / maxHR;
  if (ratio >= 0.95) return 5;
  if (ratio >= 0.9) return 4;
  if (ratio >= 0.85) return 3;
  if (ratio >= 0.78) return 2;
  return 1;
}
