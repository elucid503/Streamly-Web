import { api } from "@/api/client";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Modal } from "@/components/ui/Modal";

import type { AccessCode } from "@/lib/types";

import { Component } from "react";
import { Copy, Trash2 } from "lucide-react";

interface AdminPanelProps {

  open: boolean;

  onClose: () => void;

}

interface AdminPanelState {

  codes: AccessCode[];

  maxUses: string;
  expiresIn: string;

  loading: boolean;
  creating: boolean;

  copied: string;

}

export class AdminPanel extends Component<AdminPanelProps, AdminPanelState> {

  state: AdminPanelState = {

    codes: [],

    maxUses: "1",
    expiresIn: "168h",

    loading: false,
    creating: false,

    copied: "",

  };

  componentDidUpdate(prev: AdminPanelProps) {

    if (this.props.open && !prev.open) {

      this.load();

    }

  }

  load = async () => {

    this.setState({ loading: true });

    try {

      const codes = await api.listAccessCodes();

      this.setState({ codes, loading: false });

    } catch {

      this.setState({ loading: false });

    }

  };

  create = async () => {

    const maxUses = parseInt(this.state.maxUses, 10) || 0;

    this.setState({ creating: true });

    try {

      await api.createAccessCode(maxUses, this.state.expiresIn || undefined);

      await this.load();

    } finally {

      this.setState({ creating: false });

    }

  };

  remove = async (code: string) => {

    await api.deleteAccessCode(code);

    await this.load();

  };

  copy = (code: string) => {

    navigator.clipboard.writeText(code);

    this.setState({ copied: code });

    setTimeout(() => this.setState({ copied: "" }), 2000);

  };

  render() {

    const { open, onClose } = this.props;

    const { codes, maxUses, expiresIn, loading, creating, copied } = this.state;

    return (

      <Modal open={open} onClose={onClose} title="Access Codes" className="max-w-lg">

        <div className="mb-6 space-y-3">

          <div className="grid grid-cols-2 gap-3">

            <div>

              <label className="mb-1 block text-xs text-foreground-muted">

                Max Uses (0 = unlimited)

              </label>

              <Input value={maxUses} onChange={(e) => this.setState({ maxUses: e.target.value })} />

            </div>

            <div>

              <label className="mb-1 block text-xs text-foreground-muted">Expires In</label>

              <Input value={expiresIn}

                onChange={(e) => this.setState({ expiresIn: e.target.value })}

                placeholder="e.g. 168h"

              />

            </div>

          </div>

          <Button onClick={this.create} disabled={creating} className="w-full">

            {creating ? "Creating..." : "Generate Code"}

          </Button>

        </div>

        {loading ? (

          <div className="space-y-2">

            {Array.from({ length: 3 }).map((_, i) => (

              <div key={i} className="skeleton h-12 w-full" />

            ))}

          </div>

        ) : (

          <div className="max-h-64 space-y-2 overflow-y-auto">

            {codes.length === 0 && (
              <p className="py-4 text-center text-xs text-foreground-faint">No access codes yet</p>
            )}

            {codes.map((code) => (

              <div key={code.id} className="flex items-center justify-between rounded-md border border-border-subtle bg-surface px-3 py-2">

                <div className="min-w-0">

                  <p className="truncate font-mono text-xs">{code.code}</p>

                  <p className="text-[10px] text-foreground-faint">

                    {code.uses}
                    {code.maxUses > 0 ? ` / ${code.maxUses}` : ""} uses

                  </p>

                </div>

                <div className="flex gap-1">

                  <button onClick={() => this.copy(code.code)} className="rounded-md p-1.5 text-foreground-muted hover:bg-surface-overlay hover:text-foreground" >

                    <Copy size={13} />

                  </button>

                  <button onClick={() => this.remove(code.code)} className="rounded-md p-1.5 text-foreground-muted hover:bg-surface-overlay hover:text-red-400" >

                    <Trash2 size={13} />

                  </button>

                </div>

              </div>

            ))}

          </div>

        )}

        {copied && (

          <p className="mt-3 text-center text-xs text-foreground-muted">Copied to clipboard</p>

        )}

      </Modal>

    );

  }

}
