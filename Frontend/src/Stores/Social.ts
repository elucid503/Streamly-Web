import { TypedEmitter } from "tiny-typed-emitter";

interface SocialEvents {

  change: () => void;

}

class SocialModule extends TypedEmitter<SocialEvents> {

  incomingRequestCount = 0;
  sseEventVersion = 0;

  setIncomingRequestCount(count: number) {

    this.incomingRequestCount = count;
    this.sseEventVersion++;

    this.emit("change");

  }

}

export const social = new SocialModule();
