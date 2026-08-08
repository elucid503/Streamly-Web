import { api } from "@/api/client";

import { Modal } from "@/components/ui/Modal";
import { Switch } from "@/components/ui/Switch";

import { store } from "@/lib/store";

import { Component } from "react";

interface SettingsPanelProps {

  open: boolean;

  onClose: () => void;

}

interface SettingsPanelState {

  saving: boolean;

}

export class SettingsPanel extends Component<SettingsPanelProps, SettingsPanelState> {

  state: SettingsPanelState = { saving: false };

  update = async (patch: Partial<NonNullable<typeof store.settings>>) => {

    if (!store.settings) return;

    this.setState({ saving: true });

    try {

      const updated = await api.updateSettings(patch);

      store.setSettings(updated);

    } finally {

      this.setState({ saving: false });

    }

  };

  render() {

    const { open, onClose } = this.props;

    const settings = store.settings;

    if (!settings) return null;

    return (

      <Modal open={open} onClose={onClose} title="Settings">

        <div className="space-y-1">

          <div className="py-2">

            <label className="mb-2 block text-sm font-semibold text-foreground-muted">Preferred Quality</label>

            <div className="flex gap-2">

              {([360, 720, 1080, 2160] as const).map((h) => (

                <button key={h} className={`flex h-8 items-center rounded-md border px-3 text-xs transition-colors ${ settings.preferredHeight === h ? "border-foreground bg-foreground text-surface" : "border-border text-foreground-muted hover:text-foreground" }`}

                  onClick={() => this.update({ preferredHeight: h })}
                  disabled={this.state.saving}

                >

                  {h === 2160 ? "4K" : `${h}p`}

                </button>

              ))}

            </div>

          </div>

          <Switch

            label="Ambience lighting"
            checked={settings.ambienceEnabled}

            onChange={(v) => this.update({ ambienceEnabled: v })}

          />

          <Switch

            label="Pause overlay"
            checked={!settings.disablePauseOverlay}

            onChange={(v) => this.update({ disablePauseOverlay: !v })}

          />

          <Switch

            label="Proxy Live TV streams"
            description="Route Live TV through the server if streams are blocked."
            checked={settings.proxyLiveStreams ?? false}

            onChange={(v) => this.update({ proxyLiveStreams: v })}

          />

        </div>

      </Modal>

    );

  }

}
