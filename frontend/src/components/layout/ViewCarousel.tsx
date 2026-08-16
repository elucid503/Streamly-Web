import type { MainView } from "@/lib/types";
import { shouldReduceMotion } from "@/lib/platform";

import { Component, type ReactNode } from "react";
import { motion } from "framer-motion";

const VIEW_ORDER: MainView[] = ["vod", "live", "sports", "friends"];

interface ViewCarouselProps {

  active: MainView;

  panels: Record<MainView, ReactNode>;

}

export class ViewCarousel extends Component<ViewCarouselProps> {

  render() {

    const { active, panels } = this.props;

    const index = VIEW_ORDER.indexOf(active);

    return (

      <div className="relative w-full overflow-x-clip">

        <motion.div className="flex w-full"

          animate={{ x: `-${index * 100}%` }}
          transition={shouldReduceMotion() ? { duration: 0 } : { type: "spring", stiffness: 320, damping: 34, mass: 0.8 }}

        >

          {VIEW_ORDER.map((view) => (

            <div key={view} className="w-full flex-shrink-0">

              {panels[view]}

            </div>

          ))}

        </motion.div>

      </div>

    );

  }

}
