import { auth } from "./Auth";
import { settings } from "./Settings";
import { social } from "./Social";

const storeModules = [auth, settings, social];

export const configureStoreEmitters = (limit = 64) => {

  for (const module of storeModules) {

    if (typeof module.setMaxListeners === "function") {

      module.setMaxListeners(limit);

    }

  }

};
