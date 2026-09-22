import { TypedEmitter } from "tiny-typed-emitter";

import type { UserSettings } from "@/Features/Settings/Types";

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
