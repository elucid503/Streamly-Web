import type { UserSettings } from "@/Types";

import { TypedEmitter } from "tiny-typed-emitter";

interface SettingsEvents {

  change: () => void;

}

class SettingsModule extends TypedEmitter<SettingsEvents> {

  settings: UserSettings | null = null;

  setSettings(settings: UserSettings | null) {

    this.settings = settings;

    this.emit("change");

  }

}

export const settings = new SettingsModule();
