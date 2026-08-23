import { auth } from "./Auth";
import { configureStoreEmitters } from "./ConfigureEmitters";
import { settings } from "./Settings";
import { social } from "./Social";

export { auth as Auth, settings as Settings, social as Social };
export { configureStoreEmitters as ConfigureEmitters };

export default class Stores {

  static readonly Auth = auth;
  static readonly Settings = settings;
  static readonly Social = social;
  static readonly ConfigureEmitters = configureStoreEmitters;

}
