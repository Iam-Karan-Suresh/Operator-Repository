import { Settings as SettingsIcon, Save, User } from 'lucide-react';
import { useState, useEffect } from 'react';

const API_URL = import.meta.env.VITE_API_URL || '';

interface UISettings {
  name: string;
  profession: string;
  team: string;
}

interface SettingsProps {
  uiSettings: UISettings;
  onUpdateSettings: (s: UISettings) => void;
}

export function Settings({ uiSettings, onUpdateSettings }: SettingsProps) {
  const [localSettings, setLocalSettings] = useState<UISettings>(uiSettings);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setLocalSettings(uiSettings);
  }, [uiSettings]);

  const handleSave = async () => {
    setSaving(true);
    try {
      const resp = await fetch(`${API_URL}/api/settings`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(localSettings)
      });
      if (resp.ok) {
        const data = await resp.json();
        onUpdateSettings(data);
      }
    } catch (err) {
      console.error("Failed to save settings", err);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-500">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-foreground flex items-center">
            <SettingsIcon className="mr-3 text-primary" /> Global Settings
          </h1>
          <p className="text-muted-foreground mt-1">Manage global operator configurations and UI personalization.</p>
        </div>
        <button 
          onClick={handleSave}
          disabled={saving}
          className="px-4 py-2 bg-primary text-primary-foreground rounded-lg font-medium hover:bg-primary/90 transition-colors flex items-center disabled:opacity-50"
        >
          <Save size={16} className="mr-2" /> {saving ? 'Saving...' : 'Save Changes'}
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 mt-8">
        {/* UI Personalization */}
        <div className="glass rounded-xl p-6 border border-border">
          <h2 className="text-xl font-semibold mb-6 flex items-center">
            <User size={20} className="mr-2 text-primary" /> UI Personalization
          </h2>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-muted-foreground mb-1">Display Name</label>
              <input 
                type="text" 
                value={localSettings.name}
                onChange={(e) => setLocalSettings({...localSettings, name: e.target.value})}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50" 
                placeholder="Your Name"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-muted-foreground mb-1">Profession / Title</label>
              <input 
                type="text" 
                value={localSettings.profession}
                onChange={(e) => setLocalSettings({...localSettings, profession: e.target.value})}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50" 
                placeholder="Software Engineer"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-muted-foreground mb-1">Team Name</label>
              <input 
                type="text" 
                value={localSettings.team}
                onChange={(e) => setLocalSettings({...localSettings, team: e.target.value})}
                className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50" 
                placeholder="Platform Team"
              />
            </div>
          </div>
        </div>

        {/* Basic Settings */}
        <div className="glass rounded-xl p-6 border border-border">
          <h2 className="text-xl font-semibold mb-6 flex items-center">
            <SettingsIcon size={20} className="mr-2 text-primary" /> Basic Configuration
          </h2>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-muted-foreground mb-1">Default AWS Region</label>
              <select className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50">
                <option>us-east-1</option>
                <option>eu-central-1</option>
                <option>ap-south-1</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-muted-foreground mb-1">Default Instance Type</label>
              <select className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50">
                <option>t3.micro</option>
                <option>t3.medium</option>
                <option>m5.large</option>
              </select>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
