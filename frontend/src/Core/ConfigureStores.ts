import { auth } from "@/Features/Auth/Store";
import { settings } from "@/Features/Settings/Store";

const storeModules = [auth, settings];

export const configureStoreEmitters = (limit = 64) => {

  for (const module of storeModules) {

    if (typeof module.setMaxListeners === "function") {

      module.setMaxListeners(limit);

    }

  }

};
