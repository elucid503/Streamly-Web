import { type ReactNode } from "react";
import { createPortal } from "react-dom";
import { HelpCircle, LogOut, Menu, Search, Settings, Shield, X } from "lucide-react";
import { motion } from "framer-motion";

import { ViewContextBar, type ContextActionId } from "@/Layout/ViewContextBar";
import { Button } from "@/UI/Button";
import { Input } from "@/UI/Input";
import { Modal } from "@/UI/Modal";
import { SelectMenu } from "@/UI/SelectMenu";

import { ModuleComponent } from "@/Core/Store";
import Net from "@/Net";
import Stores from "@/Stores";
import type { FavoriteItem, MainView, WatchHistoryItem } from "@/Types";
import { cn } from "@/Utils/ClassNames";
import { navigate } from "@/Utils/Navigation";

interface SearchChromeProps {

  searchQuery: string;
  onSearch: (query: string) => void;

  view: MainView;
  showSearch: boolean;

  searchKind: "all" | "movie" | "show";
  searchYear: "all" | "2020s" | "2010s" | "2000s" | "older";
  searchRating: "all" | "7" | "8";
  searchProgress: "all" | "unwatched" | "in_progress" | "completed";

  onSearchKindChange: (value: SearchChromeProps["searchKind"]) => void;
  onSearchYearChange: (value: SearchChromeProps["searchYear"]) => void;
  onSearchRatingChange: (value: SearchChromeProps["searchRating"]) => void;
  onSearchProgressChange: (value: SearchChromeProps["searchProgress"]) => void;

  history: WatchHistoryItem[];
  favorites: FavoriteItem[];

  contextLoading: ContextActionId | null;
  onContextAction: (actionId: ContextActionId) => void;

  onOpenSettings: () => void;
  onOpenAdmin: () => void;
  onLogout: () => void;

}

interface SearchChromeState {

  faqOpen: boolean;
  version: string;
  menuPos: { top: number; left: number } | null;

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

export class SearchChrome extends ModuleComponent<SearchChromeProps, SearchChromeState> {

  state: SearchChromeState = {

    faqOpen: false,
    version: "",
    menuPos: null,

  };

  async componentDidMount() {

    this.watch(Stores.Auth);

    try {

      const { version } = await Net.Version.get();

      this.setState({ version });

    } catch {

      /* version is non-critical */

    }

  }

  openMenu = (e: React.MouseEvent<HTMLButtonElement>) => {

    e.stopPropagation();

    if (this.state.menuPos) {

      this.setState({ menuPos: null });
      return;

    }

    const rect = e.currentTarget.getBoundingClientRect();
    const menuWidth = 200;

    this.setState({

      menuPos: {

        top: rect.bottom + 6,
        left: Math.max(8, Math.min(rect.right - menuWidth, window.innerWidth - menuWidth - 8)),

      },

    });

  };

  closeMenu = () => this.setState({ menuPos: null });

  searchFilters = (): ReactNode[] => {

    const { searchKind, searchYear, searchRating, searchProgress, onSearchKindChange, onSearchYearChange, onSearchRatingChange, onSearchProgressChange } = this.props;

    return [

      <SelectMenu
        key="kind"
        label="Title type"
        value={searchKind}
        options={searchKindOptions}
        onChange={(value) => onSearchKindChange(value as SearchChromeProps["searchKind"])}
      />,

      <SelectMenu
        key="year"
        label="Release year"
        value={searchYear}
        options={searchYearOptions}
        onChange={(value) => onSearchYearChange(value as SearchChromeProps["searchYear"])}
      />,

      <SelectMenu
        key="rating"
        label="Rating"
        value={searchRating}
        options={searchRatingOptions}
        onChange={(value) => onSearchRatingChange(value as SearchChromeProps["searchRating"])}
      />,

      <SelectMenu
        key="progress"
        label="Watch progress"
        value={searchProgress}
        options={searchProgressOptions}
        onChange={(value) => onSearchProgressChange(value as SearchChromeProps["searchProgress"])}
      />,

    ];

  };

  render() {

    const {
      searchQuery,
      onSearch,
      view,
      showSearch,
      history,
      favorites,
      contextLoading,
      onContextAction,
      onOpenSettings,
      onOpenAdmin,
      onLogout,
    } = this.props;

    const { faqOpen, version, menuPos } = this.state;

    const hasQuery = searchQuery.length > 0;
    const user = Stores.Auth.user;
    const showContextBar = !showSearch && view !== "sports" && view !== "friends";

    const renderContextBar = () => (

      <ViewContextBar
        view={view}
        history={history}
        favorites={favorites}
        loadingAction={contextLoading}
        onAction={onContextAction}
      />

    );

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

        <div className={cn("flex flex-col gap-3 px-4 pt-[max(1rem,env(safe-area-inset-top))] sm:px-6 sm:pt-6 lg:px-8 lg:pt-8", view === "friends" && "lg:mx-auto lg:w-full lg:max-w-3xl")}>

          <div className="flex items-center gap-2">

            <button type="button" onClick={() => navigate("/")} className="hidden shrink-0 sm:block">

              <span className="text-sm font-semibold tracking-tight">

                Streamly <span className="font-light text-foreground-muted">Web</span>

              </span>

            </button>

            <div className="flex min-w-0 max-w-md flex-1 items-center gap-2 md:ml-2">

                <div className="relative min-w-0 flex-1">

                  <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-foreground-faint" />

                  <Input
                    className={cn("pl-9", hasQuery && "pr-9")}
                    value={searchQuery}
                    onChange={(e) => onSearch(e.target.value)}
                    placeholder={view === "live" ? "Search live TV..." : view === "sports" ? "Search sports..." : "Search movies and shows..."}
                  />

                  {hasQuery && (

                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="absolute top-1/2 right-1 -translate-y-1/2"
                      onClick={() => onSearch("")}
                      title="Clear search"
                      aria-label="Clear search"
                    >

                      <X size={16} />

                    </Button>

                  )}

                </div>

            </div>

            <div className="ml-auto flex min-w-0 shrink-0 items-center gap-2">

              {showContextBar && (

                <div className="hidden min-w-0 lg:block">

                  {renderContextBar()}

                </div>

              )}

              <Button variant="secondary" size="icon-sm" onClick={this.openMenu} title="Menu" aria-label="Menu">

                <Menu size={16} />

              </Button>

            </div>

          </div>

          {showContextBar && (

            <div className="min-w-0 lg:hidden">

              {renderContextBar()}

            </div>

          )}

          {showSearch && (

            <div className="flex flex-wrap items-center gap-1.5">

              {this.searchFilters()}

            </div>

          )}

        </div>

        {menuPos && createPortal(

          <>

            <div className="fixed inset-0 z-[99]" onClick={this.closeMenu} />

            <motion.div

              className="fixed z-[100] min-w-[200px] overflow-hidden rounded-md border border-border bg-surface-raised p-1 shadow-lg"

              style={{ top: menuPos.top, left: menuPos.left }}

              initial={{ opacity: 0, scale: 0.96, y: -6 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              transition={{ type: "spring", stiffness: 500, damping: 32 }}

            >

              <button
                type="button"
                className="flex h-9 w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left text-sm font-medium text-foreground-muted transition-colors hover:bg-surface-overlay hover:text-foreground"
                onClick={(e) => {

                  e.stopPropagation();
                  this.closeMenu();
                  onOpenSettings();

                }}
              >

                <Settings size={14} />
                <span>Settings</span>

              </button>

              {user?.isAdmin && (

                <button
                  type="button"
                  className="flex h-9 w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left text-sm font-medium text-foreground-muted transition-colors hover:bg-surface-overlay hover:text-foreground"
                  onClick={(e) => {

                    e.stopPropagation();
                    this.closeMenu();
                    onOpenAdmin();

                  }}
                >

                  <Shield size={14} />
                  <span>Admin</span>

                </button>

              )}

              <button
                type="button"
                className="flex h-9 w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left text-sm font-medium text-foreground-muted transition-colors hover:bg-surface-overlay hover:text-foreground"
                onClick={(e) => {

                  e.stopPropagation();
                  this.closeMenu();
                  this.setState({ faqOpen: true });

                }}
              >

                <HelpCircle size={14} />
                <span>Help & Legal</span>

              </button>

              <button
                type="button"
                className="flex h-9 w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left text-sm font-medium text-red-400 transition-colors hover:bg-surface-overlay hover:text-red-300"
                onClick={(e) => {

                  e.stopPropagation();
                  this.closeMenu();
                  onLogout();

                }}
              >

                <LogOut size={14} />
                <span>Log Out</span>

              </button>

              {version && (

                <div className="px-2 py-1.5 text-xs tabular-nums text-foreground-faint">

                  v{version}

                </div>

              )}

            </motion.div>

          </>,

          document.body

        )}

        {createPortal(faqModal, document.body)}

      </>

    );

  }

}
