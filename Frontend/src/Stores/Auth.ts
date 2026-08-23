import type { User } from "@/Types";

import { TypedEmitter } from "tiny-typed-emitter";

interface AuthEvents {

  change: () => void;

}

class AuthModule extends TypedEmitter<AuthEvents> {

  user: User | null = null;

  setUser(user: User | null) {

    this.user = user;

    this.emit("change");

  }

  get isAuthenticated() {

    return this.user !== null;

  }

}

export const auth = new AuthModule();
