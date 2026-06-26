import type { User, UserSettings } from "@/lib/types";

type Listener = () => void;

class AppStore {

  user: User | null = null;
  settings: UserSettings | null = null;

  incomingRequestCount = 0;
  sseEventVersion = 0;

  private listeners = new Set<Listener>();

  subscribe(listener: Listener) {

    this.listeners.add(listener);

    return () => this.listeners.delete(listener);

  }

  private notify() {

    this.listeners.forEach((l) => l());

  }

  setUser(user: User | null) {

    this.user = user;

    this.notify();

  }

  setSettings(settings: UserSettings | null) {

    this.settings = settings;

    this.notify();

  }

  setIncomingRequestCount(count: number) {

    this.incomingRequestCount = count;
    this.sseEventVersion++;

    this.notify();

  }

  get isAuthenticated() {

    return this.user !== null;

  }

}

export const store = new AppStore();
