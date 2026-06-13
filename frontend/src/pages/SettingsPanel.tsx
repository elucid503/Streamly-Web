import { Component } from "react";
import { api } from "@/api/client";
import { Modal } from "@/components/ui/Modal";
import { Switch } from "@/components/ui/Switch";
import { store } from "@/lib/store";

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
            <label className="mb-2 block text-xs text-foreground-muted">Preferred Quality</label>
            <div className="flex gap-2">
              {[720, 1080, 2160].map((h) => (
                <button
                  key={h}
                  onClick={() => this.update({ preferredHeight: h })}
                  disabled={this.state.saving}
                  className={`rounded-md border px-3 py-1.5 text-xs transition-colors ${
                    settings.preferredHeight === h
                      ? "border-foreground bg-foreground text-surface"
                      : "border-border text-foreground-muted hover:text-foreground"
                  }`}
                >
                  {h === 2160 ? "4K" : `${h}p`}
                </button>
              ))}
            </div>
          </div>

          <Switch
            label="Auto-play next episode"
            checked={settings.autoPlayNext}
            onChange={(v) => this.update({ autoPlayNext: v })}
          />
          <Switch
            label="Show skip intro"
            checked={settings.skipIntro}
            onChange={(v) => this.update({ skipIntro: v })}
          />
          <Switch
            label="Ambience lighting"
            checked={settings.ambienceEnabled}
            onChange={(v) => this.update({ ambienceEnabled: v })}
          />
          <Switch
            label="Subtitles on by default"
            checked={settings.subtitlesEnabled ?? false}
            onChange={(v) => {
              localStorage.setItem("streamly:subtitlesEnabled", v ? "1" : "0");
              void this.update({ subtitlesEnabled: v });
            }}
          />
        </div>
      </Modal>
    );
  }
}