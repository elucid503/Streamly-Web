import { Component } from "react";

import type { TypedEmitter } from "tiny-typed-emitter";

export abstract class ModuleComponent<P = object, S = object> extends Component<P, S> {

  private subs: Array<() => void> = [];

  constructor(props: P) {

    super(props);

    const previousUnmount = this.componentWillUnmount.bind(this);

    this.componentWillUnmount = () => {

      this.unsubscribeAll();
      previousUnmount();

    };

  }

  protected watch(module: TypedEmitter<{ change: () => void }>, event: "change" = "change") {

    if (typeof module.setMaxListeners === "function") {

      const current = module.getMaxListeners();

      if (current < 256) module.setMaxListeners(256);

    }

    const handler = () => this.forceUpdate();

    module.on(event, handler);
    this.subs.push(() => module.off(event, handler));

  }

  private unsubscribeAll() {

    const pending = this.subs;
    this.subs = [];

    pending.forEach((unsubscribe) => unsubscribe());

  }

  componentWillUnmount() {}

}
