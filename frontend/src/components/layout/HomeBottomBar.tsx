import { api } from "@/api/client";

import { ViewContextBar, type ContextActionId } from "@/components/layout/ViewContextBar";
import { Input } from "@/components/ui/Input";
import { Modal } from "@/components/ui/Modal";
import { SelectMenu } from "@/components/ui/SelectMenu";

import { cn } from "@/lib/utils";
import type { FavoriteItem, MainView, WatchHistoryItem } from "@/lib/types";

import { Component, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { HelpCircle, Search, X } from "lucide-react";

interface HomeBottomBarProps {

  searchQuery: string;
  onSearch: (query: string) => void;

  view: MainView;
  showSearch: boolean;

  searchKind: "all" | "movie" | "show";
  searchYear: "all" | "2020s" | "2010s" | "2000s" | "older";
  searchRating: "all" | "7" | "8";
  searchProgress: "all" | "unwatched" | "in_progress" | "completed";

  onSearchKindChange: (value: HomeBottomBarProps["searchKind"]) => void;
  onSearchYearChange: (value: HomeBottomBarProps["searchYear"]) => void;
  onSearchRatingChange: (value: HomeBottomBarProps["searchRating"]) => void;
  onSearchProgressChange: (value: HomeBottomBarProps["searchProgress"]) => void;

  history: WatchHistoryItem[];
  favorites: FavoriteItem[];

  contextLoading: ContextActionId | null;
  onContextAction: (actionId: ContextActionId) => void;

}

interface HomeBottomBarState {

  faqOpen: boolean;
  version: string;

}

const searchKindOptions = [

  { value: "all", label: "All titles" },
  { value: "show", label: "Shows" },
  { value: "movie", label: "Movies" },

];

const searchYearOptions = [

  { value: "all", label: "Any year" },
  { value: "2020s", label: "2020s" },
  { value: "2010s", label: "2010s" },
  { value: "2000s", label: "2000s" },
  { value: "older", label: "Before 2000" },

];

const searchRatingOptions = [

  { value: "all", label: "Any rating" },
  { value: "7", label: "7.0+" },
  { value: "8", label: "8.0+" },

];

const searchProgressOptions = [

  { value: "all", label: "Any progress" },
  { value: "unwatched", label: "Unwatched" },
  { value: "in_progress", label: "In progress" },
  { value: "completed", label: "Completed" },

];

const mobileSelectClass = "min-w-0 flex-1 [&_button]:h-8 [&_button]:w-full [&_button]:min-w-0 [&_button]:px-2 [&_button]:text-[11px]";

const faqItems = [

  {
    q: "What is Streamly?",
    a: "Streamly is a personal streaming aggregator that organizes and links to video content available from third-party sources across the web.",
  },
  {
    q: "Does Streamly host any content?",
    a: "No. Streamly does not host, store, or distribute any video content. All media is streamed directly from independent third-party sources. It is solely an interface that indexes and links to existing publicly available streams.",
  },
  {
    q: "Copyright & DMCA",
    a: "Streamly does not control or upload any of the content accessible through this service. If you are a rights holder and believe content is being made available inappropriately, please contact the relevant hosting provider directly. Streamly is not a host and cannot remove content from third-party servers.",
  },
  {
    q: "Is this service legal?",
    a: "Streamly operates similarly to a search engine — it does not provide, upload, or profit from any copyrighted content. Responsibility for the legality of accessing third-party streams rests with the end user and their local jurisdiction.",
  },
  {
    q: "Privacy & Data",
    a: "Streamly only stores what is necessary to operate the service: your watch history and favorites. No personally identifiable data is shared with or sold to any third party.",
  },

];

export class HomeBottomBar extends Component<HomeBottomBarProps, HomeBottomBarState> {

  state: HomeBottomBarState = {

    faqOpen: false,
    version: "",

  };

  async componentDidMount() {

    try {

      const { version } = await api.getVersion();

      this.setState({ version });

    } catch {

      /* version is non-critical */

    }

  }

  searchFilters = (compact = false): ReactNode[] => {

    const { searchKind, searchYear, searchRating, searchProgress, onSearchKindChange, onSearchYearChange, onSearchRatingChange, onSearchProgressChange, } = this.props;

    const selectClass = compact ? mobileSelectClass : undefined;

    return [

      <SelectMenu
        key="kind"
        className={selectClass}
        label="Title type"
        value={searchKind}
        options={searchKindOptions}
        onChange={(value) => onSearchKindChange(value as HomeBottomBarProps["searchKind"])}
      />,

      <SelectMenu
        key="year"
        className={selectClass}
        label="Release year"
        value={searchYear}
        options={searchYearOptions}
        onChange={(value) => onSearchYearChange(value as HomeBottomBarProps["searchYear"])}
      />,

      <SelectMenu
        key="rating"
        className={selectClass}
        label="Rating"
        value={searchRating}
        options={searchRatingOptions}
        onChange={(value) => onSearchRatingChange(value as HomeBottomBarProps["searchRating"])}
      />,

      <SelectMenu
        key="progress"
        className={selectClass}
        label="Watch progress"
        value={searchProgress}
        options={searchProgressOptions}
        onChange={(value) => onSearchProgressChange(value as HomeBottomBarProps["searchProgress"])}
      />,

    ];

  };

  renderSearchFilters = (side: "left" | "right") => {

    const filters = this.searchFilters();
    const splitAt = Math.ceil(filters.length / 2);
    const sideFilters = side === "left" ? filters.slice(0, splitAt) : filters.slice(splitAt);

    if (sideFilters.length === 0) return null;

    return (

      <div className={cn(

        "flex min-w-0 flex-wrap items-center gap-2",
        side === "left" ? "justify-center lg:justify-end" : "justify-center lg:justify-start"

      )}>

        {sideFilters}

      </div>

    );

  };

  renderSearchField = (compact = false) => {

    const { searchQuery, onSearch } = this.props;
    const hasQuery = searchQuery.length > 0;

    return (

      <div className={cn("relative w-full", compact ? "mx-auto max-w-none" : "mx-auto max-w-md")}>

        <Search size={compact ? 14 : 16} className="absolute top-1/2 left-3 -translate-y-1/2 text-foreground-faint lg:left-4" />

        <Input className={cn(

            "w-full rounded-full pl-9 lg:pl-11",
            compact ? "h-9" : "h-10",
            hasQuery && "pr-9 lg:pr-11"

          )}

          value={searchQuery}
          onChange={(e) => onSearch(e.target.value)}
          placeholder="Search titles..."

        />

        {hasQuery && (

          <button type="button"
            onClick={() => onSearch("")}
            aria-label="Clear search"

            className="absolute top-1/2 right-3 -translate-y-1/2 rounded-full p-0.5 text-foreground-faint transition-colors hover:text-foreground lg:right-4"

          >

            <X size={compact ? 14 : 16} />

          </button>

        )}

      </div>

    );

  };

  renderMobileActions = () => {

    const { view, showSearch, history, favorites, contextLoading, onContextAction } = this.props;

    if (showSearch) {

      return (

        <div className="flex w-full gap-1 pb-0.5">

          {this.searchFilters(true)}

        </div>

      );

    }

    return (

      <ViewContextBar

        compact
        view={view}
        history={history}
        favorites={favorites}

        loadingAction={contextLoading}
        onAction={onContextAction}

      />

    );

  };

  render() {

    const { view, showSearch, history, favorites, contextLoading, onContextAction } = this.props;
    const { faqOpen, version } = this.state;

    const faqModal = (
      <Modal open={faqOpen} onClose={() => this.setState({ faqOpen: false })} title="Help & Legal">

        <div className="space-y-4 text-sm">

          {faqItems.map((item) => (

            <div key={item.q}>

              <p className="mb-1 font-medium text-foreground">{item.q}</p>

              <p className="leading-relaxed text-foreground-muted">{item.a}</p>

            </div>

          ))}

        </div>

      </Modal>
    );

    return (

      <>

        <div className="fixed inset-x-0 bottom-0 z-40 border-t border-border-subtle bg-surface/80 backdrop-blur-md pb-[max(0.25rem,calc(env(safe-area-inset-bottom,0px)*0.55))] lg:pb-0">

          <div className="mx-auto flex max-w-[1600px] flex-col gap-1.5 px-3 py-2 lg:hidden">

            {this.renderSearchField(true)}

            {this.renderMobileActions()}

          </div>

          <div className="mx-auto hidden w-full max-w-[1600px] grid-cols-[1fr_auto_1fr] items-center gap-3 px-4 py-2.5 sm:px-8 lg:grid">

            <div className="min-w-0">

              {showSearch ? this.renderSearchFilters("left") : (

                <ViewContextBar

                  view={view}
                  side="left"
                  history={history}
                  favorites={favorites}

                  loadingAction={contextLoading}
                  onAction={onContextAction}

                />

              )}

            </div>

            <div className="w-full lg:w-auto">

              {this.renderSearchField()}

            </div>

            <div className="min-w-0">

              {showSearch ? this.renderSearchFilters("right") : (

                <ViewContextBar

                  view={view}
                  side="right"
                  history={history}
                  favorites={favorites}

                  loadingAction={contextLoading}
                  onAction={onContextAction}

                />

              )}

            </div>

          </div>

          {/* Absolute-positioned corner elements - desktop only */}

          <button

            type="button"
            aria-label="Help & Legal"
            title="Help & Legal"

            className="absolute bottom-4 left-4 hidden items-center justify-center rounded-full p-1 text-foreground-faint transition-colors hover:text-foreground-muted sm:left-8 lg:flex"

            onClick={() => this.setState({ faqOpen: true })}

          >

            <HelpCircle size={18} />

          </button>

          {version && (

            <span className="absolute bottom-4 right-4 hidden text-[13px] tabular-nums text-foreground-faint sm:right-8 lg:block">

              v{version}

            </span>

          )}

        </div>

        {createPortal(faqModal, document.body)}

      </>

    );

  }

}
