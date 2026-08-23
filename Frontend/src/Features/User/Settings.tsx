import { Modal } from "@/UI/Modal";
import { Switch } from "@/UI/Switch";

import { ModuleComponent } from "@/Core/Store";
import Net from "@/Net";
import Stores from "@/Stores";
import type { UserSettings } from "@/Types";

interface SettingsPanelProps {

  open: boolean;

  onClose: () => void;

}

interface SettingsPanelState {

  saving: boolean;

}

export class SettingsPanel extends ModuleComponent<SettingsPanelProps, SettingsPanelState> {

  state: SettingsPanelState = { saving: false };

  componentDidMount() {

    this.watch(Stores.Settings);

  }

  update = async (patch: Partial<UserSettings>) => {

    if (!Stores.Settings.settings) return;

    this.setState({ saving: true });

    try {

      const updated = await Net.Settings.update(patch);

      Stores.Settings.setSettings(updated);

    } finally {

      this.setState({ saving: false });

    }

  };

  render() {

    const { open, onClose } = this.props;

    const settings = Stores.Settings.settings;

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

            label="Ambience Lighting"
            description="Uses the current frame to fill empty space."

            checked={settings.ambienceEnabled}

            onChange={(v) => this.update({ ambienceEnabled: v })}

          />

          <Switch

            label="Pause Overlay"
            description="Show details when pausing the Player."

            checked={!settings.disablePauseOverlay}

            onChange={(v) => this.update({ disablePauseOverlay: !v })}

          />

          <Switch

            label="Proxy Live TV"
            description="Route Live TV through the server if you can't watch."
            checked={settings.proxyLiveStreams ?? false}

            onChange={(v) => this.update({ proxyLiveStreams: v })}

          />

          <Switch

            label="Hide Ad Breaks"
            description="Cover and mute most ad breaks on Live TV."
            checked={settings.detectLiveAds ?? false}

            onChange={(v) => this.update({ detectLiveAds: v })}

          />

        </div>

      </Modal>

    );

  }

}
