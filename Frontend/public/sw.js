self.addEventListener("install", (event) => {

  event.waitUntil(self.skipWaiting());

});

self.addEventListener("activate", (event) => {

  event.waitUntil(self.clients.claim());

});

self.addEventListener("push", (event) => {

  let payload = {};

  try {

    payload = event.data ? event.data.json() : {};

  } catch {

    payload = { body: event.data ? event.data.text() : "" };

  }

  const title = payload.title || "Streamly";
  const body = payload.body || "A game you follow is starting.";
  const url = payload.url || "/";
  const tag = payload.tag || "sports";

  event.waitUntil(

    self.registration.showNotification(title, {

      body,
      icon: "/icon-192.png",
      badge: "/icon-192.png",
      tag,
      data: { url },
      renotify: false,

    })

  );

});

self.addEventListener("notificationclick", (event) => {

  event.notification.close();

  const raw = event.notification.data && event.notification.data.url ? event.notification.data.url : "/";
  const target = new URL(raw, self.location.origin);
  const path = target.pathname + target.search + target.hash;

  event.waitUntil(openOrFocus(path));

});

async function openOrFocus(path) {

  const windows = await self.clients.matchAll({ type: "window", includeUncontrolled: true });

  for (const client of windows) {

    if (typeof client.url === "string" && client.url.startsWith(self.location.origin)) {

      client.postMessage({ type: "navigate", url: path });

      if ("focus" in client) {

        await client.focus();

      }

      return;

    }

  }

  await self.clients.openWindow(path);

}
