import { cn } from "@/lib/utils";

import { Component, type ReactNode } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { X } from "lucide-react";

interface ModalProps {

  title: string;

  open: boolean;
  onClose: () => void;

  children: ReactNode;
  className?: string;

}

export class Modal extends Component<ModalProps> {

  render() {

    const { open, onClose, title, children, className } = this.props;

    return (

      <AnimatePresence>

        {open && (

          <motion.div className="fixed inset-0 z-50 flex items-center justify-center p-4"

            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}

            exit={{ opacity: 0 }}

          >

            <div className="absolute inset-0 bg-surface/60 backdrop-blur-md" onClick={onClose} />

            <motion.div className={cn(

                "relative z-10 w-full max-w-md rounded-lg border border-border-subtle bg-surface/80 p-6 shadow-2xl backdrop-blur-md",
                className

              )}

              initial={{ opacity: 0, scale: 0.96, y: 8 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}

              exit={{ opacity: 0, scale: 0.96, y: 8 }}

              transition={{ type: "spring", stiffness: 400, damping: 30 }}

            >

              <div className="mb-5 flex items-center justify-between">

                <h2 className="text-base font-medium">

                  {title}

                </h2>

                <button onClick={onClose} className="rounded-md p-1 text-foreground-muted transition-colors hover:bg-surface-overlay hover:text-foreground" >

                  <X size={16} />

                </button>

              </div>

              {children}

            </motion.div>

          </motion.div>

        )}

      </AnimatePresence>

    );

  }

}
