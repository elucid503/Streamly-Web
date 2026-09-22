import { pushAPI } from "@/Features/Sports/PushApi";
import { sportsAPI } from "@/Features/Sports/Api";
import { versionAPI } from "@/Core/VersionApi";
import { ApiError } from "@/Core/Request";
import { iosNeedsInstallForPush, pushSupported } from "@/Utils/Platform";

export type AlertHint = "install" | "denied" | "unsupported" | "unavailable";

export type SubscribeResult = { ok: true } | { ok: false; hint: AlertHint };

let workerPromise: Promise<ServiceWorkerRegistration | null> | null = null;

function urlBase64ToUint8Array(base64: string): Uint8Array {

  const padding = "=".repeat((4 - (base64.length % 4)) % 4);
  const normalized = (base64 + padding).replace(/-/g, "+").replace(/_/g, "/");
  const raw = atob(normalized);
  const output = new Uint8Array(raw.length);

  for (let i = 0; i < raw.length; i++) {

    output[i] = raw.charCodeAt(i);

  }

  return output;

}

export async function registerSportsWorker(): Promise<ServiceWorkerRegistration | null> {

  if (!pushSupported()) {

    return null;

  }

  if (!workerPromise) {

    workerPromise = (async () => {

      let version = "1";

      try {

        const data = await versionAPI.get();

        if (data.version) version = data.version;

      } catch {

        /* cache-bust with a fallback */

      }

      return navigator.serviceWorker.register(`/sw.js?v=${encodeURIComponent(version)}`, {

        updateViaCache: "none",

      });

    })().catch((err) => {

      workerPromise = null;
      console.warn("sports alerts: service worker register failed", err);

      return null;

    });

  }

  return workerPromise;

}

async function ensurePushSubscription(): Promise<PushSubscription> {

  const registration = await registerSportsWorker();

  if (!registration) {

    throw new Error("unsupported");

  }

  const existing = await registration.pushManager.getSubscription();

  if (existing) {

    await persistSubscription(existing);

    return existing;

  }

  const { publicKey } = await pushAPI.vapidKey();
  const key = urlBase64ToUint8Array(publicKey);

  const created = await registration.pushManager.subscribe({

    userVisibleOnly: true,
    applicationServerKey: key.buffer as ArrayBuffer,

  });

  await persistSubscription(created);

  return created;

}

async function persistSubscription(subscription: PushSubscription) {

  const json = subscription.toJSON();
  const endpoint = json.endpoint ?? subscription.endpoint;
  const p256dh = json.keys?.p256dh;
  const auth = json.keys?.auth;

  if (!endpoint || !p256dh || !auth) {

    throw new Error("invalid subscription");

  }

  await pushAPI.upsertSubscription({

    endpoint,
    keys: { p256dh, auth },

  });

}

export async function subscribeToMatch(matchId: string): Promise<SubscribeResult> {

  if (iosNeedsInstallForPush()) {

    return { ok: false, hint: "install" };

  }

  if (!pushSupported()) {

    return { ok: false, hint: "unsupported" };

  }

  if (Notification.permission === "denied") {

    return { ok: false, hint: "denied" };

  }

  if (Notification.permission !== "granted") {

    const permission = await Notification.requestPermission();

    if (permission !== "granted") {

      return { ok: false, hint: permission === "denied" ? "denied" : "unsupported" };

    }

  }

  try {

    await ensurePushSubscription();
    await sportsAPI.subscribe(matchId);

    return { ok: true };

  } catch (err) {

    if (err instanceof ApiError && err.status === 503) {

      return { ok: false, hint: "unavailable" };

    }

    if (err instanceof Error && err.message === "unsupported") {

      return { ok: false, hint: "unsupported" };

    }

    throw err;

  }

}

export async function unsubscribeFromMatch(matchId: string): Promise<void> {

  await sportsAPI.unsubscribe(matchId);

}

export async function subscribeToTeam(team: string): Promise<SubscribeResult> {

  if (iosNeedsInstallForPush()) {

    return { ok: false, hint: "install" };

  }

  if (!pushSupported()) {

    return { ok: false, hint: "unsupported" };

  }

  if (Notification.permission === "denied") {

    return { ok: false, hint: "denied" };

  }

  if (Notification.permission !== "granted") {

    const permission = await Notification.requestPermission();

    if (permission !== "granted") {

      return { ok: false, hint: permission === "denied" ? "denied" : "unsupported" };

    }

  }

  try {

    await ensurePushSubscription();
    await sportsAPI.subscribeTeam(team);

    return { ok: true };

  } catch (err) {

    if (err instanceof ApiError && err.status === 503) {

      return { ok: false, hint: "unavailable" };

    }

    if (err instanceof Error && err.message === "unsupported") {

      return { ok: false, hint: "unsupported" };

    }

    throw err;

  }

}

export async function unsubscribeFromTeam(team: string): Promise<void> {

  await sportsAPI.unsubscribeTeam(team);

}

export function hintCopy(hint: AlertHint): string {

  switch (hint) {

    case "install":

      return "Add Streamly to your Home Screen to get alerts on iPhone.";

    case "denied":

      return "Notifications are blocked for this site. Enable them in the browser to get alerts.";

    case "unsupported":

      return "This browser cannot receive alerts.";

    case "unavailable":

      return "Alerts are not configured on the server yet.";

  }

}
