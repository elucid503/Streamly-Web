import { cn } from "@/lib/utils";

import { Component, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";

interface ModalProps {

  title: string;

  open: boolean;
  onClose: () => void;

  children: ReactNode;
  className?: string;

}

export class Modal extends Component<ModalProps> {

  private openedAt = 0;

  componentDidMount() {

    if (this.props.open) this.openedAt = Date.now();

  }

  componentDidUpdate(prev: ModalProps) {

    if (this.props.open && !prev.open) this.openedAt = Date.now();

  }

  closeFromBackdrop = () => {

    // The opening tap can land on this overlay once it mounts under the finger.
    if (Date.now() - this.openedAt < 400) return;

    this.props.onClose();

  };

  render() {

    const { open, onClose, title, children, className } = this.props;

    if (!open || typeof document === "undefined") return null;

    return createPortal(

      <div className="fixed inset-0 z-[110] flex items-center justify-center p-4">

        <div className="absolute inset-0 bg-surface/60 backdrop-blur-md" onClick={this.closeFromBackdrop} />

        <div className={cn(

            "relative z-10 w-full max-w-md rounded-xl border border-border bg-surface-raised p-6 shadow-2xl",
            className

          )}

        >

          <div className="mb-5 flex items-center justify-between">

            <h2 className="text-base font-semibold">

              {title}

            </h2>

            <button onClick={onClose} className="rounded-md p-1.5 text-foreground-muted transition-colors hover:bg-surface-overlay hover:text-foreground" >

              <X size={16} />

            </button>

          </div>

          {children}

        </div>

      </div>,

      document.body,

    );

  }

}
