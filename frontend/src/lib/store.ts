import type { User, UserSettings } from "@/lib/types";

type Listener = () => void;

class AppStore {

  user: User | null = null;
  settings: UserSettings | null = null;

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

  get isAuthenticated() {

    return this.user !== null;

  }

}

export const store = new AppStore();
