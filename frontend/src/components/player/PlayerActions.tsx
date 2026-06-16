import { cn } from "@/lib/utils";

import { useEffect, useLayoutEffect, useState, type AnimationEvent } from "react";
import { ChevronLeft, ChevronRight, Volume1, Volume2 } from "lucide-react";

export type PlayerActionFeedback = | { id: number; kind: "seek"; direction: -1 | 1; label: string } | { id: number; kind: "volume"; direction: -1 | 1; label: string };

interface PlayerActionFeedbackOverlayProps {

  feedback: PlayerActionFeedback | null;

}

export function PlayerActionFeedbackOverlay({ feedback }: PlayerActionFeedbackOverlayProps) {

  const [displayed, setDisplayed] = useState<PlayerActionFeedback | null>(null);
  const [phase, setPhase] = useState<"enter" | "exit">("enter");
  const [enterReady, setEnterReady] = useState(false);

  useEffect(() => {

    if (feedback) {

      setDisplayed(feedback);

      setPhase("enter");

      return;

    }

    setPhase("exit");

    setEnterReady(false);

  }, [feedback]);

  useLayoutEffect(() => {

    if (phase !== "enter" || !displayed) {

      setEnterReady(false);

      return;

    }

    setEnterReady(false);

    let raf2 = 0;

    const raf1 = requestAnimationFrame(() => {

      raf2 = requestAnimationFrame(() => setEnterReady(true));

    });

    return () => {

      cancelAnimationFrame(raf1);

      cancelAnimationFrame(raf2);

    };

  }, [displayed?.id, phase]);

  const handleAnimationEnd = (event: AnimationEvent<HTMLDivElement>) => {

    if (phase !== "exit" || event.animationName !== "player-feedback-out") return;

    setDisplayed(null);
    setPhase("enter");
    setEnterReady(false);

  };

  if (!displayed) return null;

  return (

    <div className={cn(

        "pointer-events-none absolute top-1/2 z-30 -translate-y-1/2",
        displayed.kind === "seek" ? displayed.direction < 0 ? "left-6 sm:left-12" : "right-6 sm:right-12" : "left-1/2 -translate-x-1/2"

      )}

    >

      <div className={cn(

          "flex items-center gap-1.5 rounded-md border border-border-subtle bg-surface/80 px-3 py-1.5 text-sm font-medium text-foreground/95 backdrop-blur-md",

          enterReady && phase === "enter" && "player-feedback-enter",
          phase === "exit" && "player-feedback-exit"

        )}

        onAnimationEnd={handleAnimationEnd}

      >

        {displayed.kind === "seek" ? (

          displayed.direction < 0 ? (

            <>

              <ChevronLeft size={18} strokeWidth={2.5} />
              <span className="tabular-nums">{displayed.label}</span>

            </>

          ) : (

            <>

              <span className="tabular-nums">{displayed.label}</span>
              <ChevronRight size={18} strokeWidth={2.5} />

            </>

          )

        ) : displayed.direction < 0 ? (

          <>

            <Volume1 size={16} />
            <span className="tabular-nums">{displayed.label}</span>

          </>

        ) : (

          <>

            <Volume2 size={16} />
            <span className="tabular-nums">{displayed.label}</span>

          </>

        )}

      </div>

    </div>

  );

}
