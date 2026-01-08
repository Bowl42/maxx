import { useState } from 'react';
import { Settings, Plus, Save, Trash2, X, Moon, Sun, Monitor } from 'lucide-react';
import { useSettings, useUpdateSetting, useDeleteSetting } from '@/hooks/queries';
import { useTheme, type Theme } from '@/hooks/use-theme';

export function SettingsPage() {
  return (
    <div className="flex flex-col h-full">
      <Header />
      <div className="flex-1 overflow-y-auto p-lg">
        <div className="max-w-2xl mx-auto space-y-8">
          <AppearanceSection />
          <SystemSettingsSection />
        </div>
      </div>
    </div>
  );
}

function Header() {
  return (
    <div className="h-[73px] flex items-center gap-md p-lg border-b border-border bg-surface-primary">
      <div className="w-10 h-10 rounded-lg bg-accent/10 flex items-center justify-center">
        <Settings size={20} className="text-accent" />
      </div>
      <div>
        <h1 className="text-headline font-semibold text-text-primary">Settings</h1>
        <p className="text-caption text-text-secondary">Configure your maxx-next instance</p>
      </div>
    </div>
  );
}

function AppearanceSection() {
  const { theme, setTheme } = useTheme();

  const themes: { value: Theme; label: string; icon: typeof Sun }[] = [
    { value: 'light', label: 'Light', icon: Sun },
    { value: 'dark', label: 'Dark', icon: Moon },
    { value: 'system', label: 'System', icon: Monitor },
  ];

  return (
    <section>
      <h2 className="text-title3 font-semibold text-text-primary mb-4">Appearance</h2>
      <div className="bg-surface-secondary rounded-lg p-4">
        <label className="text-sm font-medium text-text-primary block mb-3">Theme</label>
        <div className="flex gap-2">
          {themes.map(({ value, label, icon: Icon }) => (
            <button
              key={value}
              onClick={() => setTheme(value)}
              className={`flex items-center gap-2 px-4 py-2 rounded-lg border transition-colors ${
                theme === value
                  ? 'border-accent bg-accent/10 text-accent'
                  : 'border-border bg-surface-primary text-text-secondary hover:bg-surface-hover'
              }`}
            >
              <Icon size={16} />
              <span className="text-sm">{label}</span>
            </button>
          ))}
        </div>
      </div>
    </section>
  );
}

function SystemSettingsSection() {
  const { data: settings, isLoading } = useSettings();
  const updateSetting = useUpdateSetting();
  const deleteSetting = useDeleteSetting();
  const [newKey, setNewKey] = useState('');
  const [newValue, setNewValue] = useState('');
  const [showAddForm, setShowAddForm] = useState(false);
  const [editingKey, setEditingKey] = useState<string | null>(null);
  const [editValue, setEditValue] = useState('');

  const handleAdd = () => {
    if (!newKey.trim()) return;
    updateSetting.mutate(
      { key: newKey.trim(), value: newValue },
      {
        onSuccess: () => {
          setNewKey('');
          setNewValue('');
          setShowAddForm(false);
        },
      }
    );
  };

  const handleEdit = (key: string) => {
    if (!settings) return;
    setEditingKey(key);
    setEditValue(settings[key] || '');
  };

  const handleSave = (key: string) => {
    updateSetting.mutate(
      { key, value: editValue },
      {
        onSuccess: () => {
          setEditingKey(null);
          setEditValue('');
        },
      }
    );
  };

  const handleDelete = (key: string) => {
    if (confirm(`Delete setting "${key}"?`)) {
      deleteSetting.mutate(key);
    }
  };

  const settingsEntries = settings ? Object.entries(settings) : [];

  return (
    <section>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-title3 font-semibold text-text-primary">System Settings</h2>
        <button
          onClick={() => setShowAddForm(true)}
          className="btn btn-primary flex items-center gap-2"
        >
          <Plus size={14} />
          Add Setting
        </button>
      </div>

      <div className="bg-surface-secondary rounded-lg overflow-hidden">
        {isLoading ? (
          <div className="p-8 text-center text-text-muted">Loading settings...</div>
        ) : settingsEntries.length === 0 && !showAddForm ? (
          <div className="p-8 text-center text-text-muted">
            <Settings size={32} className="mx-auto mb-2 opacity-30" />
            <p>No custom settings configured</p>
          </div>
        ) : (
          <div className="divide-y divide-border">
            {showAddForm && (
              <div className="p-4 bg-accent/5">
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={newKey}
                    onChange={(e) => setNewKey(e.target.value)}
                    placeholder="Key"
                    className="form-input flex-1"
                    autoFocus
                  />
                  <input
                    type="text"
                    value={newValue}
                    onChange={(e) => setNewValue(e.target.value)}
                    placeholder="Value"
                    className="form-input flex-1"
                  />
                  <button
                    onClick={handleAdd}
                    disabled={!newKey.trim() || updateSetting.isPending}
                    className="btn btn-primary"
                  >
                    <Save size={14} />
                  </button>
                  <button
                    onClick={() => {
                      setShowAddForm(false);
                      setNewKey('');
                      setNewValue('');
                    }}
                    className="btn bg-surface-hover text-text-secondary"
                  >
                    <X size={14} />
                  </button>
                </div>
              </div>
            )}
            {settingsEntries.map(([key, value]) => (
              <div key={key} className="p-4 flex items-center gap-4">
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium text-text-primary font-mono">{key}</div>
                  {editingKey === key ? (
                    <input
                      type="text"
                      value={editValue}
                      onChange={(e) => setEditValue(e.target.value)}
                      className="form-input mt-1 w-full"
                      autoFocus
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') handleSave(key);
                        if (e.key === 'Escape') setEditingKey(null);
                      }}
                    />
                  ) : (
                    <div
                      className="text-sm text-text-secondary font-mono truncate cursor-pointer hover:text-text-primary"
                      onClick={() => handleEdit(key)}
                    >
                      {value || <span className="italic text-text-muted">(empty)</span>}
                    </div>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  {editingKey === key ? (
                    <>
                      <button
                        onClick={() => handleSave(key)}
                        disabled={updateSetting.isPending}
                        className="btn btn-primary"
                      >
                        <Save size={14} />
                      </button>
                      <button
                        onClick={() => setEditingKey(null)}
                        className="btn bg-surface-hover text-text-secondary"
                      >
                        <X size={14} />
                      </button>
                    </>
                  ) : (
                    <button
                      onClick={() => handleDelete(key)}
                      className="btn bg-error/10 text-error hover:bg-error/20"
                    >
                      <Trash2 size={14} />
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </section>
  );
}

export default SettingsPage;
